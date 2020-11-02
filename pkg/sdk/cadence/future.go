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

package cadence

import (
	"go.uber.org/cadence/workflow"
)

// NewReadyFuture returns a ready future object with its value and error set to
// the specified values.
func NewReadyFuture(ctx workflow.Context, value interface{}, err error) workflow.Future {
	future, settable := workflow.NewFuture(ctx)
	settable.Set(value, err)

	return future
}
