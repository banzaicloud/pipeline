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

package providers

import (
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers/alibaba"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle"
)

const (
	Alibaba = alibaba.Provider
	Amazon  = amazon.Provider
	Azure   = azure.Provider
	Google  = google.Provider
	Oracle  = oracle.Provider
)

// ValidateProvider validates if the passed cloud provider is supported.
// Unsupported cloud providers trigger an pkgErrors.ErrorNotSupportedCloudType error.
func ValidateProvider(provider string) error {
	switch provider {
	case Alibaba:
	case Amazon:
	case Google:
	case Azure:
	case Oracle:
	default:
		// TODO: create an error value in this package instead
		return pkgErrors.ErrorNotSupportedCloudType
	}

	return nil
}
