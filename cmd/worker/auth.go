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

package main

import (
	bvsdkauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"go.uber.org/cadence/worker"

	authworkflow "github.com/banzaicloud/pipeline/src/auth/workflow"
)

// registerCronWorkflows registers the domain specific cron workflows
func registerAuthWorkflows(worker worker.Worker, tokenStore bvsdkauth.TokenStore) (err error) {
	authworkflow.NewStartTokenStoreGCActivity(tokenStore).Register(worker)
	authworkflow.NewGarbageCollectTokenStoreWorkflow().Register(worker)

	return nil
}
