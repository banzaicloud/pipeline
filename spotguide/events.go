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

package spotguide

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
)

type authEvents interface {
	NotifyOrganizationRegistered(fn interface{})
}

type eventBus interface {
	SubscribeAsync(topic string, fn interface{}, transactional bool) error
}

type ebAuthEvents struct {
	eb eventBus
}

func (e ebAuthEvents) NotifyOrganizationRegistered(fn interface{}) {
	e.eb.SubscribeAsync(auth.OrganizationRegisteredTopic, fn, false) // nolint: errcheck
}

var AuthEventEmitter authEvents = ebAuthEvents{config.EventBus} // nolint: gochecknoglobals
