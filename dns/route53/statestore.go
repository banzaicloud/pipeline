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

import pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"

// awsRoute53StateStore manages the state of the domains
// registered by us in the Amazon Route53 external DNS service
type awsRoute53StateStore interface {
	create(state *domainState) error
	update(state *domainState) error
	find(orgId pkgAuth.OrganizationID, domain string, state *domainState) (bool, error)
	findByStatus(status string) ([]domainState, error)
	findByOrgId(orgId pkgAuth.OrganizationID, state *domainState) (bool, error)
	listUnused() ([]domainState, error)
	delete(state *domainState) error
}
