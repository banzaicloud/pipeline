// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
