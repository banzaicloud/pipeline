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

package route53

// DomainEvent holds the common fields for the domain events
type DomainEvent struct {
	Domain         string
	OrganisationId uint
}

// RegisterDomainSucceededEvent is fired when a domain is registered or re-registered in an external DNS service
type RegisterDomainSucceededEvent struct {
	DomainEvent
}

// RegisterDomainFailedEvent is fired when a domain registration or re-registration in an external DNS service
// failed
type RegisterDomainFailedEvent struct {
	DomainEvent
	Cause error
}

// UnregisterDomainSucceededEvent is fired when a domain is un-registered in an external DNS service
type UnregisterDomainSucceededEvent struct {
	DomainEvent
}

// UnregisterDomainFailedEvent is fired when a domain un-registered in an external DNS service
// failed
type UnregisterDomainFailedEvent struct {
	DomainEvent
	Cause error
}
