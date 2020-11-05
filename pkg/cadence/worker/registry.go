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
//
// Copyright (c) 2017-2020 Uber Technologies Inc.
// Portions of the Software are attributed to Copyright (c) 2020 Temporal Technologies Inc.

package worker

import (
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

// Registry is a subset of the Worker interface to only expose registration functions to consumers.
type Registry interface {
	WorkflowRegistry
	ActivityRegistry
}

// WorkflowRegistry is a subset of the Worker interface to only expose workflow registration functions to consumers.
type WorkflowRegistry interface {
	// RegisterWorkflow - registers a workflow function with the worker.
	// A workflow takes a workflow.Context and input and returns a (result, error) or just error.
	// Examples:
	//	func sampleWorkflow(ctx workflow.Context, input []byte) (result []byte, err error)
	//	func sampleWorkflow(ctx workflow.Context, arg1 int, arg2 string) (result []byte, err error)
	//	func sampleWorkflow(ctx workflow.Context) (result []byte, err error)
	//	func sampleWorkflow(ctx workflow.Context, arg1 int) (result string, err error)
	// Serialization of all primitive types, structures is supported ... except channels, functions, variadic, unsafe pointer.
	// For global registration consider workflow.Register
	// This method panics if workflowFunc doesn't comply with the expected format or tries to register the same workflow
	RegisterWorkflow(w interface{})

	// RegisterWorkflowWithOptions registers the workflow function with options.
	// The user can use options to provide an external name for the workflow or leave it empty if no
	// external name is required. This can be used as
	//  worker.RegisterWorkflowWithOptions(sampleWorkflow, RegisterWorkflowOptions{})
	//  worker.RegisterWorkflowWithOptions(sampleWorkflow, RegisterWorkflowOptions{Name: "foo"})
	// This method panics if workflowFunc doesn't comply with the expected format or tries to register the same workflow
	// type name twice. Use workflow.RegisterOptions.DisableAlreadyRegisteredCheck to allow multiple registrations.
	RegisterWorkflowWithOptions(w interface{}, options workflow.RegisterOptions)
}

// ActivityRegistry is a subset of the Worker interface to only expose activity registration functions to consumers.
type ActivityRegistry interface {
	// RegisterActivity - register an activity function or a pointer to a structure with the worker.
	// An activity function takes a context and input and returns a (result, error) or just error.
	//
	// And activity struct is a structure with all its exported methods treated as activities. The default
	// name of each activity is the method name.
	//
	// Examples:
	//	func sampleActivity(ctx context.Context, input []byte) (result []byte, err error)
	//	func sampleActivity(ctx context.Context, arg1 int, arg2 string) (result *customerStruct, err error)
	//	func sampleActivity(ctx context.Context) (err error)
	//	func sampleActivity() (result string, err error)
	//	func sampleActivity(arg1 bool) (result int, err error)
	//	func sampleActivity(arg1 bool) (err error)
	//
	//  type Activities struct {
	//     // fields
	//  }
	//  func (a *Activities) SampleActivity1(ctx context.Context, arg1 int, arg2 string) (result *customerStruct, err error) {
	//    ...
	//  }
	//
	//  func (a *Activities) SampleActivity2(ctx context.Context, arg1 int, arg2 *customerStruct) (result string, err error) {
	//    ...
	//  }
	//
	// Serialization of all primitive types, structures is supported ... except channels, functions, variadic, unsafe pointer.
	// This method panics if activityFunc doesn't comply with the expected format or an activity with the same
	// type name is registered more than once.
	// For global registration consider activity.Register
	RegisterActivity(a interface{})

	// RegisterActivityWithOptions registers the activity function or struct pointer with options.
	// The user can use options to provide an external name for the activity or leave it empty if no
	// external name is required. This can be used as
	//  worker.RegisterActivityWithOptions(barActivity, RegisterActivityOptions{})
	//  worker.RegisterActivityWithOptions(barActivity, RegisterActivityOptions{Name: "barExternal"})
	// When registering the structure that implements activities the name is used as a prefix that is
	// prepended to the activity method name.
	//  worker.RegisterActivityWithOptions(&Activities{ ... }, RegisterActivityOptions{Name: "MyActivities_"})
	// To override each name of activities defined through a structure register the methods one by one:
	// activities := &Activities{ ... }
	// worker.RegisterActivityWithOptions(activities.SampleActivity1, RegisterActivityOptions{Name: "Sample1"})
	// worker.RegisterActivityWithOptions(activities.SampleActivity2, RegisterActivityOptions{Name: "Sample2"})
	// See RegisterActivity function for more info.
	// The other use of options is to disable duplicated activity registration check
	// which might be useful for integration tests.
	// worker.RegisterActivityWithOptions(barActivity, RegisterActivityOptions{DisableAlreadyRegisteredCheck: true})
	RegisterActivityWithOptions(a interface{}, options activity.RegisterOptions)
}
