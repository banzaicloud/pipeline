// Copyright © 2021 Banzai Cloud
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

package workflow

import (
	"context"

	"emperror.dev/errors"
	bvsdkauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

// StartTokenStoreGCActivityName is the name of the token store garbage
// collection starter activity.
const StartTokenStoreGCActivityName = "auth-start-token-store-gc"

// StartTokenStoreGCActivity collects the necessary component dependencies
// for executing a token store garbage collection starting operation.
type StartTokenStoreGCActivity struct {
	tokenStore bvsdkauth.TokenStore
}

// NewStartTokenStoreGCActivity instantiates a activity object for starting the
// token store garbage collection.
func NewStartTokenStoreGCActivity(tokenStore bvsdkauth.TokenStore) *StartTokenStoreGCActivity {
	return &StartTokenStoreGCActivity{
		tokenStore: tokenStore,
	}
}

// Execute executes a token store garbage collection starting operation.
func (a *StartTokenStoreGCActivity) Execute(ctx context.Context) (err error) {
	err = a.tokenStore.GC()
	if err != nil {
		return errors.Wrap(err, "failed to garbage collect TokenStore")
	}

	return nil
}

// Register registers the token store garbage collection starting activity.
func (a StartTokenStoreGCActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: StartTokenStoreGCActivityName})
}

// startTokenStoreGC starts the token store garbage collection and returns an
// error if any occurs.
//
// This is a convenience wrapper around the corresponding activity.
func startTokenStoreGC(ctx workflow.Context) error {
	return startTokenStoreGCAsync(ctx).Get(ctx, nil)
}

// startTokenStoreGCAsync returns a future object for starting the token store
// garbage collection.
//
// This is a convenience wrapper around the corresponding activity.
func startTokenStoreGCAsync(ctx workflow.Context) workflow.Future {
	return workflow.ExecuteActivity(ctx, StartTokenStoreGCActivityName)
}
