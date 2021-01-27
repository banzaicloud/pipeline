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

const instancePipelineCreatedTag = "pipeline-created"

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

// IsPipelineRelated checks whether the spot request has a tag which indicates that it was created by Pipeline
func (r *SpotInstanceRequest) IsPipelineRelated() bool {
	if len(r.Tags) == 0 {
		return false
	}

	for _, tag := range r.Tags {
		if tag.Key != nil && tag.Value != nil && *tag.Key == instancePipelineCreatedTag && *tag.Value == "true" {
			return true
		}
	}

	return false
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
	case "pending-evaluation", "not-scheduled-yet", "pending-fulfillment", "capacity-not-available", "capacity-oversubscribed":
		return true
	}

	return false
}

// GetState gives back the state of the request
func (r *SpotInstanceRequest) GetState() string {
	var state string
	if r.State != nil {
		state = *r.State
	}
	return state
}

// GetStatusCode gives back the status code of the request
func (r *SpotInstanceRequest) GetStatusCode() string {
	var status string
	if r.Status != nil && r.Status.Code != nil {
		status = *r.Status.Code
	}
	return status
}
