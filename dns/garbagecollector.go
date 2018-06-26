package dns

import (
	"time"
)

// garbageCollector is the interface for domain garbage collector implementations.
// The garbage collector's responsibility to clean up unused domains from external
// DNS service
type garbageCollector interface {
	// Starts the garbage collector
	start() error
	// Stops the garbage collector
	stop()
}

// garbageCollector cleans up unused domains registered in external DNS service
type garbageCollectorImpl struct {
	gcInterval       time.Duration
	ticker           *time.Ticker
	dnsServiceClient DnsServiceClient
}

func (gc *garbageCollectorImpl) start() error {
	gc.ticker = time.NewTicker(gc.gcInterval)

	go func() {
		for range gc.ticker.C {
			log.Debug("DNS garbage collector running")
			gc.dnsServiceClient.Cleanup()
		}
	}()

	return nil
}

func (gc *garbageCollectorImpl) stop() {
	gc.ticker.Stop()
}

// newGarbageCollector creates a garbage collector for domains managed by the given
// external DNS service
func newGarbageCollector(dnsServiceClient DnsServiceClient, gcInterval time.Duration) (garbageCollector, error) {
	return &garbageCollectorImpl{gcInterval: gcInterval, dnsServiceClient: dnsServiceClient}, nil
}
