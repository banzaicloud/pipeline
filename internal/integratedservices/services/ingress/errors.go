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

package ingress

import (
	"fmt"

	pkgerrors "github.com/banzaicloud/pipeline/pkg/errors"
)

type unsupportedControllerError struct {
	Controller string

	pkgerrors.ValidationBehavior
}

func (e unsupportedControllerError) Error() string {
	return fmt.Sprintf("controller %q is not supported", e.Controller)
}

type unavailableControllerError struct {
	Controller string

	pkgerrors.BadRequestBehavior
	pkgerrors.ClientErrorBehavior
	pkgerrors.ValidationBehavior
}

func (e unavailableControllerError) Error() string {
	return fmt.Sprintf("controller %q is currently not available", e.Controller)
}

type unsupportedServiceTypeError struct {
	ServiceType string

	pkgerrors.BadRequestBehavior
	pkgerrors.ClientErrorBehavior
	pkgerrors.ValidationBehavior
}

func (e unsupportedServiceTypeError) Error() string {
	return fmt.Sprintf("service type %q is not supported", e.ServiceType)
}
