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

package auth

import pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"

// OrganizationRegisteredTopic is the name of the topic where organization registration events are published.
const OrganizationRegisteredTopic = "organization_registered"

// authEvents is responsible for dispatching domain events throughout the system.
// It does not express any infrastructural detail (like pubsub).
type authEvents interface {
	OrganizationRegistered(organizationID uint, userID pkgAuth.UserID)
}

type eventBus interface {
	Publish(topic string, args ...interface{})
}

type ebAuthEvents struct {
	eb eventBus
}

func (e ebAuthEvents) OrganizationRegistered(organizationID uint, userID pkgAuth.UserID) {
	e.eb.Publish(OrganizationRegisteredTopic, organizationID, userID)
}
