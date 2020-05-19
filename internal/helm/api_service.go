// Copyright © 2020 Banzai Cloud
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

package helm

import (
	"context"
)

// HelmAPI interface intended to gather all the helm operations exposed as rest services
type RestAPI interface {
	// Gets releases and decorates the re
	GetReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) (releaseList []DetailedRelease, err error)
}

// DetailedRelease wraps a release and adds additional information to it
type DetailedRelease struct {
	Release
	Supported bool `json:"supported"`
}
