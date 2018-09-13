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

package cluster

import (
	"net/http"

	"google.golang.org/api/googleapi"
)

type resourceChecker interface {
	getType() string
	list() ([]string, error)
	isResourceDeleted(string) error
	forceDelete(string) error
}

type resourceCheckers []resourceChecker

const (
	firewall       = "firewall"
	forwardingRule = "forwardingRule"
	targetPool     = "targetPool"
)

// isResourceNotFound transforms an error into googleapi.Error
func isResourceNotFound(err error) error {
	apiError, isOk := err.(*googleapi.Error)
	if isOk {
		if apiError.Code == http.StatusNotFound {
			return nil
		}
	}
	return err
}
