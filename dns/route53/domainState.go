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

import (
	"time"
)

// status
const (
	CREATING = "CREATING"
	CREATED  = "CREATED"
	FAILED   = "FAILED"
	REMOVING = "REMOVING"
)

// domainState represents the state of a domain registered with Amazon Route53 DNS service
type domainState struct {
	createdAt      time.Time
	organisationId uint
	domain         string
	hostedZoneId   string
	policyArn      string
	iamUser        string
	awsAccessKeyId string
	status         string
	errMsg         string
}
