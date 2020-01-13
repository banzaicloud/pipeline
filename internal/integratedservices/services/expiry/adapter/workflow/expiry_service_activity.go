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

package workflow

import (
	"context"
)

const ExpireActivityName = "expire-cluster-activity"

type ExpiryActivityInput struct {
	ClusterID uint
}

type ExpiryActivity struct {
	clusterDeleter ClusterDeleter
}

// ClusterDeleter contract for triggering cluster deletion.
// Designed to be used by the expire integrated service
type ClusterDeleter interface {
	Delete(ctx context.Context, clusterID uint) error
}

func NewExpiryActivity(clusterDeleter ClusterDeleter) ExpiryActivity {
	return ExpiryActivity{
		clusterDeleter: clusterDeleter,
	}
}

func (a ExpiryActivity) Execute(ctx context.Context, input ExpiryActivityInput) error {

	return a.clusterDeleter.Delete(ctx, input.ClusterID)
}
