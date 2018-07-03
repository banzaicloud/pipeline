package dns

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns/route53"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"sync"
	"time"
)

var log *logrus.Logger

var once sync.Once
var errCreate error

// dnsServiceClient is the  DnsServiceClient singleton instance if this functionality is enabled
var dnsServiceClient DnsServiceClient

var gc garbageCollector

// Simple init for logging
func init() {
	log = config.Logger()
}

// DnsServiceClient contains the operations for managing domains in a Dns Service
type DnsServiceClient interface {
	RegisterDomain(orgId uint, domain string) error
	UnregisterDomain(orgId uint, domain string) error
	IsDomainRegistered(orgId uint, domain string) (bool, error)
	Cleanup()
}

type externalDnsServiceClientSync struct {
	// muxOrgDomain is a mutex used to sync access to external Dns service
	muxOrgDomain sync.Mutex

	dnsSvcClientImpl DnsServiceClient
}

// RegisterDomain registers a domain in external DNS service
func (dns *externalDnsServiceClientSync) RegisterDomain(orgId uint, domain string) error {
	dns.muxOrgDomain.Lock()
	defer dns.muxOrgDomain.Unlock()

	return dns.dnsSvcClientImpl.RegisterDomain(orgId, domain)
}

// UnregisterDomain unregisters a domain in external DNS service
func (dns *externalDnsServiceClientSync) UnregisterDomain(orgId uint, domain string) error {
	dns.muxOrgDomain.Lock()
	defer dns.muxOrgDomain.Unlock()

	return dns.dnsSvcClientImpl.UnregisterDomain(orgId, domain)
}

// IsDomainRegistered returns true if domain is registered in external DNS service
func (dns *externalDnsServiceClientSync) IsDomainRegistered(orgId uint, domain string) (bool, error) {
	dns.muxOrgDomain.Lock()
	defer dns.muxOrgDomain.Unlock()

	return dns.dnsSvcClientImpl.IsDomainRegistered(orgId, domain)
}

// Cleanup cleans up unused domains
func (dns *externalDnsServiceClientSync) Cleanup() {
	dns.muxOrgDomain.Lock()
	defer dns.muxOrgDomain.Unlock()

	dns.dnsSvcClientImpl.Cleanup()

}

func newExternalDnsServiceClientInstance() {
	dnsServiceClient = nil
	errCreate = nil

	gcInterval := time.Duration(viper.GetInt("dns.gcIntervalMinute")) * time.Minute

	// This is how the secrets are expected to be written in Vault:
	// vault kv put secret/banzaicloud/aws AWS_REGION=... AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=...
	awsCredentialsPath := viper.GetString("aws.credentials.path")

	//var secret *api.Secret
	secret, err := secret.Store.Logical.Read(awsCredentialsPath)
	if err != nil {
		log.Errorf("Failed to read AWS credentials from Vault: %s", err.Error())
		errCreate = err
		return
	}

	if secret == nil {
		log.Infoln("No AWS credentials for Route53 provided in Vault")
		return
	}

	awsCredentials := cast.ToStringMapString(secret.Data["data"])
	region := awsCredentials[secretTypes.AwsRegion]
	awsSecretId := awsCredentials[secretTypes.AwsAccessKeyId]
	awsSecretKey := awsCredentials[secretTypes.AwsSecretAccessKey]

	if len(region) == 0 || len(awsSecretId) == 0 || len(awsSecretKey) == 0 {
		log.Infoln("No AWS credentials for Route53 provided in Vault")
		return
	}

	awsRoute53, err := route53.NewAwsRoute53(region, awsSecretId, awsSecretKey)
	if err != nil {
		errCreate = err
		return
	}

	dnsServiceClient = &externalDnsServiceClientSync{dnsSvcClientImpl: awsRoute53}

	garbageCollector, err := newGarbageCollector(dnsServiceClient, gcInterval)

	if err != nil {
		errCreate = err
		return
	}

	gc = garbageCollector
	if err := gc.start(); err != nil {
		errCreate = err
	}
}

// GetExternalDnsServiceClient creates a new external dns service client
func GetExternalDnsServiceClient() (DnsServiceClient, error) {

	// create a singleton
	once.Do(func() {
		newExternalDnsServiceClientInstance()
	})

	return dnsServiceClient, errCreate

}
