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

package awssdk

import (
	"context"

	"go.uber.org/cadence/workflow"
)

type contextKey string

func (c contextKey) String() string {
	return "cadence_aws_" + string(c)
}

const (
	// ContextSecretID identifies a secret ID in a context.
	ContextSecretID = contextKey("secretID")

	// ContextRegion identifies a region in a context.
	ContextRegion = contextKey("region")
)

// WithSecretID returns a new context with a secret ID.
func WithSecretID(ctx workflow.Context, secretID string) workflow.Context {
	return workflow.WithValue(ctx, ContextSecretID, secretID)
}

// SecretID returns a secret ID from a context (if any).
func SecretID(ctx context.Context) (string, bool) {
	secretID, ok := ctx.Value(ContextSecretID).(string)

	return secretID, ok
}

// WithRegion returns a new context with a secret ID.
func WithRegion(ctx workflow.Context, secretID string) workflow.Context {
	return workflow.WithValue(ctx, ContextRegion, secretID)
}

// Region returns a secret ID from a context (if any).
func Region(ctx context.Context) (string, bool) {
	secretID, ok := ctx.Value(ContextRegion).(string)

	return secretID, ok
}
