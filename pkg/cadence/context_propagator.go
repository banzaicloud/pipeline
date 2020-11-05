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
	"context"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
)

// StringContextPropagator implements a custom workflow.ContextPropagator
// for passing a single string value to workflows and activities.
type StringContextPropagator struct {
	// PropagationKey identifies a value in Cadence message headers.
	PropagationKey string

	// ContextKey identifies values in a context.
	ContextKey interface{}

	// Optional values will not emit an error when the key is missing from the context.
	// Useful when you want to propagate handling missing values to a higher level.
	Optional bool
}

func (s StringContextPropagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	secretID, ok := ctx.Value(s.ContextKey).(string)
	if !ok && !s.Optional {
		return errors.Errorf("unable to extract key from context %v", s.ContextKey)
	}

	if !ok {
		return nil
	}

	writer.Set(s.PropagationKey, []byte(secretID))

	return nil
}

func (s StringContextPropagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	secretID, ok := ctx.Value(s.ContextKey).(string)
	if !ok && !s.Optional {
		return errors.Errorf("unable to extract key from context %v", s.ContextKey)
	}

	if !ok {
		return nil
	}

	writer.Set(s.PropagationKey, []byte(secretID))

	return nil
}

func (s StringContextPropagator) Extract(ctx context.Context, reader workflow.HeaderReader) (context.Context, error) {
	if err := reader.ForEachKey(func(key string, value []byte) error {
		if key == s.PropagationKey {
			ctx = context.WithValue(ctx, s.ContextKey, string(value))

			return nil
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (s StringContextPropagator) ExtractToWorkflow(ctx workflow.Context, reader workflow.HeaderReader) (workflow.Context, error) {
	if err := reader.ForEachKey(func(key string, value []byte) error {
		if key == s.PropagationKey {
			ctx = workflow.WithValue(ctx, s.ContextKey, string(value))

			return nil
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ctx, nil
}
