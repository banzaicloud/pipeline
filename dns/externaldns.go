package dns

import (
	"sync"
	"time"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns/route53"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/satori/go.uuid"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

var once sync.Once
var errCreate error

// dnsServiceClient is the DnsServiceClient singleton instance if this functionality is enabled
var dnsServiceClient DnsServiceClient

var gc garbageCollector

// dnsNotificationsChannel is used to receive DNS related events from Route53 and fan out the events to consumers.
var dnsNotificationsChannel chan interface{}

// dnsEventsConsumers stores the channels through which subscribers receive DNS events
var dnsEventsConsumers map[uuid.UUID]chan<- interface{}

var mux sync.RWMutex

// DnsEventsSubscription represents a subscription to Dns events
type DnsEventsSubscription struct {
	Id     uuid.UUID
	Events <-chan interface{}
}

// DnsServiceClient contains the operations for managing domains in a Dns Service
type DnsServiceClient interface {
	RegisterDomain(orgId uint, domain string) error
	UnregisterDomain(orgId uint, domain string) error
	IsDomainRegistered(orgId uint, domain string) (bool, error)
	Cleanup()
	ProcessUnfinishedTasks()
}

func newExternalDnsServiceClientInstance() {
	dnsServiceClient = nil
	errCreate = nil

	gcInterval := time.Duration(viper.GetInt(config.DNSGcIntervalMinute)) * time.Minute

	// This is how the secrets are expected to be written in Vault:
	// vault kv put secret/banzaicloud/aws AWS_REGION=... AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=...
	awsCredentialsPath := viper.GetString(config.AwsCredentialPath)

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

	dnsNotificationsChannel = make(chan interface{})
	awsRoute53, err := route53.NewAwsRoute53(region, awsSecretId, awsSecretKey, dnsNotificationsChannel)
	if err != nil {
		errCreate = err

		close(dnsNotificationsChannel)
		return
	}
	dnsServiceClient = awsRoute53

	// initiate and start DNS garbage collector
	garbageCollector, err := newGarbageCollector(dnsServiceClient, gcInterval)

	if err != nil {
		errCreate = err
		close(dnsNotificationsChannel)
		return
	}

	gc = garbageCollector
	if err := gc.start(); err != nil {
		close(dnsNotificationsChannel)
		errCreate = err
		return
	}

	dnsEventsConsumers = make(map[uuid.UUID]chan<- interface{})

	// start DNS events observer
	go observeDnsEvents()

	// process in progress domain registration/un-registration
	dnsServiceClient.ProcessUnfinishedTasks()
}

// GetExternalDnsServiceClient creates a new external dns service client
func GetExternalDnsServiceClient() (DnsServiceClient, error) {

	// create a singleton
	once.Do(func() {
		newExternalDnsServiceClientInstance()
	})

	return dnsServiceClient, errCreate

}

// SubscribeDnsEvents returns DnsEventsSubscription to caller.
// The subscriber can receive DNS domain related events from
// the Events field of the subscription
func SubscribeDnsEvents() *DnsEventsSubscription {
	if dnsServiceClient == nil {
		return nil
	}

	mux.Lock()
	defer mux.Unlock()

	eventsChannel := make(chan interface{})
	subscription := DnsEventsSubscription{
		Id:     uuid.NewV4(),
		Events: eventsChannel,
	}

	dnsEventsConsumers[subscription.Id] = eventsChannel

	return &subscription
}

// UnsubscribeDnsEvents deletes the subscription with the given id
func UnsubscribeDnsEvents(id uuid.UUID) {
	if dnsServiceClient == nil {
		return
	}

	mux.Lock()
	defer mux.Unlock()

	if eventsChannel, ok := dnsEventsConsumers[id]; ok {
		delete(dnsEventsConsumers, id)

		close(eventsChannel)
	}

}

func observeDnsEvents() {
	if dnsServiceClient == nil || dnsNotificationsChannel == nil {
		return
	}

	for event := range dnsNotificationsChannel {
		log.Debugf("DNS event observer: received event %v", event)
		notifySubscribers(event)
	}
}

func notifySubscribers(event interface{}) {
	mux.RLock()
	defer mux.RUnlock()

	log.Debugf("DNS event observer: publishing event %v to subscribers", event)
	for _, eventsChannel := range dnsEventsConsumers {
		eventsChannel <- event
	}
}
