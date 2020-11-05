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

package pkeworkflow

import (
	"context"
	"crypto/sha1"
	"fmt"

	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const CalculateNodePoolVersionActivityName = "pke-calculate-node-pool-version"

// CalculateNodePoolVersionActivity calculates the node pool version.
type CalculateNodePoolVersionActivity struct{}

type CalculateNodePoolVersionActivityInput struct {
	Image      string
	VolumeSize int
	Version    string
}

type CalculateNodePoolVersionActivityOutput struct {
	Version string
}

// NewCalculateNodePoolVersionActivity creates a new CalculateNodePoolVersionActivity instance.
func NewCalculateNodePoolVersionActivity() CalculateNodePoolVersionActivity {
	return CalculateNodePoolVersionActivity{}
}

// Register registers the activity in the worker.
func (a CalculateNodePoolVersionActivity) Register(worker worker.ActivityRegistry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: CalculateNodePoolVersionActivityName})
}

// Execute is the main body of the activity.
func (a CalculateNodePoolVersionActivity) Execute(
	_ context.Context,
	input CalculateNodePoolVersionActivityInput,
) (CalculateNodePoolVersionActivityOutput, error) {
	return CalculateNodePoolVersionActivityOutput{
		Version: calculateNodePoolVersion(
			input.Image,
			fmt.Sprint(input.VolumeSize),
			input.Version,
		),
	}, nil
}

func calculateNodePoolVersion(input ...string) string {
	h := sha1.New() // #nosec

	for _, i := range input {
		_, _ = h.Write([]byte(i))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
