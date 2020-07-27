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

package globaleks

import (
	"sync"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

// nolint: gochecknoglobals
var imageSelector eks.ImageSelector

// nolint: gochecknoglobals
var imageSelectorMu sync.Mutex

// ImageSelector returns an initialized ImageSelector instance.
func ImageSelector() eks.ImageSelector {
	imageSelectorMu.Lock()
	defer imageSelectorMu.Unlock()

	return imageSelector
}

// SetImageSelector configures an ImageSelector instance.
func SetImageSelector(is eks.ImageSelector) {
	imageSelectorMu.Lock()
	defer imageSelectorMu.Unlock()

	imageSelector = is
}
