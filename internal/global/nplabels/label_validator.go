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

package nplabels

import (
	"sync"
)

// nolint: gochecknoglobals
var nplLabelValidator LabelValidator

// nolint: gochecknoglobals
var nplLabelValidatorMu sync.Mutex

type LabelValidator interface {
	ValidateKey(key string) error
	ValidateValue(value string) error
	ValidateLabel(key string, value string) error
	ValidateLabels(labels map[string]string) error
}

// NodePoolLabelValidator returns a global node pool validator.
func NodePoolLabelValidator() LabelValidator {
	nplLabelValidatorMu.Lock()
	defer nplLabelValidatorMu.Unlock()

	return nplLabelValidator
}

// SetNodePoolLabelValidator configures a global node pool validator.
func SetNodePoolLabelValidator(v LabelValidator) {
	nplLabelValidatorMu.Lock()
	defer nplLabelValidatorMu.Unlock()

	nplLabelValidator = v
}
