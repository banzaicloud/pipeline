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

package ec2

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

// SpotInstanceRequest extends ec2.SpotInstanceRequest
type SpotInstanceRequest struct {
	*ec2.SpotInstanceRequest
}

// NewSpotInstanceRequest initialises and gives back a SpotInstanceRequest
func NewSpotInstanceRequest(request *ec2.SpotInstanceRequest) *SpotInstanceRequest {

	return &SpotInstanceRequest{
		SpotInstanceRequest: request,
	}
}

// IsActive is true if the request is an active one
func (r *SpotInstanceRequest) IsActive() bool {
	state := r.GetState()

	return state == ec2.SpotInstanceStateOpen || state == ec2.SpotInstanceStateActive
}

// IsFullfilled is true if the request is fulfilled
func (r *SpotInstanceRequest) IsFulfilled() bool {
	return r.GetStatusCode() == "fulfilled"
}

// IsPending is true if the request is in a pending state
func (r *SpotInstanceRequest) IsPending() bool {
	if !r.IsActive() {
		return false
	}

	switch r.GetStatusCode() {
	case "pending-evaluation", "not-scheduled-yet", "pending-fulfillment":
		return true
	}

	return false
}

// GetState gives back the state of the request
func (r *SpotInstanceRequest) GetState() string {
	return *r.State
}

// GetStatusCode gives back the status code of the request
func (r *SpotInstanceRequest) GetStatusCode() string {
	return *r.Status.Code
}
