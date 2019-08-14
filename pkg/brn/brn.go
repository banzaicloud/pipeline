// Copyright © 2019 Banzai Cloud
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

package brn

import (
	"strconv"
	"strings"

	"emperror.dev/errors"
)

// Scheme is the scheme of the resource name format.
const Scheme = "brn"

// SchemePrefix can be used to check if a string is a BRN.
const SchemePrefix = Scheme + ":"

// Resource type constants
const (
	SecretResourceType = "secret"
)

// ErrInvalid is returned when a BRN fails validation checks.
var ErrInvalid = errors.NewPlain("brn: invalid BRN")

// ErrUnexpectedResourceType is returned when a BRN cannot be parsed as a specific resource type.
var ErrUnexpectedResourceType = errors.NewPlain("brn: unexpected resource type")

// ResourceName represents a Banzai Resource Name.
// String form: `brn:organizationID:resourceType:resourceID`
type ResourceName struct {
	// Scheme is always "brn". Placed here for future compatibility.
	Scheme string

	// OrganizationID is the ID of the organization that owns the resource.
	// This can be ignored when the resource itself can be identified
	// (eg. when the resource is unique).
	OrganizationID uint

	// ResourceType identifies the type of the resource.
	// Eg. secret
	ResourceType string

	// ResourceID is the identifier of the resource.
	ResourceID string
}

// String returns the original BRN.
func (n ResourceName) String() string {
	orgID := ""
	if n.OrganizationID != 0 {
		orgID = strconv.FormatUint(uint64(n.OrganizationID), 10)
	}

	components := []string{
		n.Scheme,
		orgID,
		n.ResourceType,
		n.ResourceID,
	}

	return strings.Join(components, ":")
}

// IsBRN checks if the supplied string looks like a BRN.
func IsBRN(s string) bool {
	return strings.HasPrefix(s, SchemePrefix)
}

// Parse accepts a BRN parses it into a ResourceName.
func Parse(brn string) (ResourceName, error) {
	const rnLen = 4
	components := strings.SplitN(brn, ":", rnLen)

	if len(components) < rnLen {
		return ResourceName{}, errors.WithStack(ErrInvalid)
	}

	orgID := uint(0)
	if components[1] != "" {
		o, err := strconv.ParseUint(components[1], 10, 64)
		if err != nil {
			return ResourceName{}, errors.Wrap(err, "invalid organization ID")
		}

		orgID = uint(o)
	}

	return ResourceName{
		Scheme:         components[0],
		OrganizationID: orgID,
		ResourceType:   components[2],
		ResourceID:     components[3],
	}, nil
}

// ParseAs function parses a BRN into a resource name and checks if the resource is of type resourceType
func ParseAs(brn string, resourceType string) (ResourceName, error) {
	rn, err := Parse(brn)
	if err != nil {
		return rn, err
	}

	if rn.ResourceType != resourceType {
		return rn, errors.WithDetails(
			errors.WithStack(ErrUnexpectedResourceType),
			"resourceType", resourceType,
		)
	}

	return rn, err
}
