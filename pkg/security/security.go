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

package security

import (
	"time"
)

type Policy struct {
	Id      string       `json:"id"`
	Name    string       `json:"name"`
	Comment string       `json:"comment,omitempty"`
	Version string       `json:"version"`
	Rules   []PolicyRule `json:"rules"`
}

// A bundle containing a set of policies, whitelists, and rules for mapping them to specific images
type PolicyBundle struct {
	// Id of the bundle
	Id string `json:"id"`
	// Human readable name for the bundle
	Name string `json:"name"`
	// Description of the bundle, human readable
	Comment string `json:"comment"`
	// Version id for this bundle format
	Version string `json:"version,omitempty"`
	// Whitelists which define which policy matches to disregard explicitly in the final policy decision
	Whitelists []Whitelist `json:"whitelists"`
	// Policies which define the go/stop/warn status of an image using rule matches on image properties
	Policies []Policy `json:"policies"`
	// Mapping rules for defining which policy and whitelist(s) to apply to an image based on a match of the image tag or id. Evaluated in order.
	Mappings []MappingRule `json:"mappings"`
	// List of mapping rules that define which images should always be passed (unless also on the blacklist), regardless of policy result.
	WhitelistedImages []ImageSelectionRule `json:"whitelisted_images,omitempty"`
	// List of mapping rules that define which images should always result in a STOP/FAIL policy result regardless of policy content or presence in whitelisted_images
	BlacklistedImages []ImageSelectionRule `json:"blacklisted_images,omitempty"`
}

type MappingRule struct {
	Id           string   `json:"id,omitempty"`
	Name         string   `json:"name"`
	WhitelistIds []string `json:"whitelist_ids"`
	// Optional single policy to evalute, if set will override any value in policy_ids, for backwards compatibility. Generally, policy_ids should be used even with a array of length 1.
	PolicyId string `json:"policy_id,omitempty"`
	// List of policyIds to evaluate in order, to completion
	PolicyIds  []string `json:"policy_ids,omitempty"`
	Registry   string   `json:"registry"`
	Repository string   `json:"repository"`
	Image      ImageRef `json:"image"`
}

type ImageSelectionRule struct {
	Id         string   `json:"id,omitempty"`
	Name       string   `json:"name"`
	Registry   string   `json:"registry"`
	Repository string   `json:"repository"`
	Image      ImageRef `json:"image"`
}

// A reference to an image
type ImageRef struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// A policy bundle plus some metadata
type PolicyBundleRecord struct {
	CreatedAt   time.Time `json:"created_at,omitempty"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
	// The bundle's identifier
	PolicyId string `json:"policyId,omitempty"`
	// True if the bundle is currently defined to be used automatically
	Active bool `json:"active,omitempty"`
	// UserId of the user that owns the bundle
	UserId string `json:"userId,omitempty"`
	// Source location of where the policy bundle originated
	PolicySource string       `json:"policy_source,omitempty"`
	Policybundle PolicyBundle `json:"policybundle,omitempty"`
}

// A rule that defines and decision value if the match is found true for a given image.
type PolicyRule struct {
	Id      string             `json:"id"`
	Gate    string             `json:"gate"`
	Trigger string             `json:"trigger"`
	Action  string             `json:"action"`
	Params  []PolicyRuleParams `json:"params"`
}

type PolicyRuleParams struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// A collection of whitelist items to match a policy evaluation against.
type Whitelist struct {
	Id      string          `json:"id"`
	Name    string          `json:"name,omitempty"`
	Version string          `json:"version"`
	Comment string          `json:"comment,omitempty"`
	Items   []WhitelistItem `json:"items,omitempty"`
}

// Identifies a specific gate and trigger match from a policy against an image and indicates it should be ignored in final policy decisions
type WhitelistItem struct {
	Id        string `json:"id,omitempty"`
	Gate      string `json:"gate"`
	TriggerId string `json:"trigger_id"`
}

type ReleaseWhiteListItem struct {
	Name   string `json:"name" binding:"required"`
	Owner  string `json:"owner" binding:"required"`
	Reason string `json:"reason"`
	Regexp string `json:"regexp,omitempty"`
}
