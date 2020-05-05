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

package processlog

import (
	"go.uber.org/cadence/workflow"
)

// ProcessLogger keeps track of long-running processes.
type ProcessLogger interface {
	// StartProcess records the beginning of a process.
	StartProcess(ctx workflow.Context, typ string, resourceID string)
}

// Process is a long-running job/workflow/whatever that includes activities.
type Process interface {
	// Finish records the end of a process.
	Finish(ctx workflow.Context, err error)

	// StartActivity records a new activity of a process.
	StartActivity(ctx workflow.Context, typ string) Activity
}

// Activity is a short lived part of a Process.
type Activity interface {
	// Finish records the end of a process.
	Finish(ctx workflow.Context, err error)
}
