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

package autoscaling

import "fmt"

type errCouldBeFinal interface {
	IsFinal() bool
}

// IsErrorFinal checks whether the error is final
func IsErrorFinal(err error) bool {
	e, ok := err.(errCouldBeFinal)
	if !ok {
		return false
	}

	return e.IsFinal()
}

type errAutoscalingGroupNotHealthy struct {
	Desired int
	Actual  int
}

// NewAutoscalingGroupNotHealthyError creates a new errAutoscalingGroupNotHealthy
func NewAutoscalingGroupNotHealthyError(desired int, actual int) error {
	return errAutoscalingGroupNotHealthy{
		Desired: desired,
		Actual:  actual,
	}
}

func (e errAutoscalingGroupNotHealthy) Error() string {
	return fmt.Sprintf("ASG is not in desired state (desired: %d, actual: %d)", e.Desired, e.Actual)
}
func (e errAutoscalingGroupNotHealthy) IsFinal() bool { return false }
func (e errAutoscalingGroupNotHealthy) Context() []interface{} {
	return []interface{}{"desired", e.Desired, "actual", e.Actual}
}
