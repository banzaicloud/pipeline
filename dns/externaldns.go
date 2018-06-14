package dns

import "github.com/banzaicloud/pipeline/dns/route53"

// DnsServiceClient contains the operations for managing domains in a Dns Service
type DnsServiceClient interface {
	RegisterDomain(uint, string) error
	UnregisterDomain(uint, string) error
	IsDomainRegistered(uint, string) (bool, error)
}

// NewExternalDnsServiceClient creates a new external dns service client
func NewExternalDnsServiceClient(region, awsSecretId, awsSecretKey string) (DnsServiceClient, error) {
	return route53.NewAwsRoute53(region, awsSecretId, awsSecretKey)
}
