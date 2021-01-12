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

package awsworkflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/testsuite"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

func TestCalculateNodePoolVersionActivity(t *testing.T) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()

	NewCalculateNodePoolVersionActivity().Register(env)

	input := CalculateNodePoolVersionActivityInput{
		Image: "ami-xxxxxxxxxxxxx",
		VolumeEncryption: &eks.NodePoolVolumeEncryption{
			Enabled:          true,
			EncryptionKeyARN: "arn:aws:kms:region:account:key/id",
		},
		VolumeSize: 50,
		CustomSecurityGroups: []string{
			"sg-1",
			"sg-2",
		},
	}

	v, err := env.ExecuteActivity(CalculateNodePoolVersionActivityName, input)
	require.NoError(t, err)

	var output CalculateNodePoolVersionActivityOutput

	err = v.Get(&output)
	require.NoError(t, err)

	assert.Equal(
		t,
		CalculateNodePoolVersionActivityOutput{
			Version: eks.CalculateNodePoolVersion(
				input.Image,
				fmt.Sprintf("%v", *input.VolumeEncryption),
				fmt.Sprintf("%d", input.VolumeSize),
				strings.Join(input.CustomSecurityGroups, ","),
			),
		},
		output,
	)

	input2 := CalculateNodePoolVersionActivityInput{}

	v, err = env.ExecuteActivity(CalculateNodePoolVersionActivityName, input2)
	require.NoError(t, err)

	var output2 CalculateNodePoolVersionActivityOutput

	err = v.Get(&output2)
	require.NoError(t, err)

	assert.Equal(
		t,
		CalculateNodePoolVersionActivityOutput{
			Version: eks.CalculateNodePoolVersion(
				input2.Image,
				"<nil>",
				fmt.Sprintf("%d", input2.VolumeSize),
				strings.Join(input2.CustomSecurityGroups, ","),
			),
		},
		output2,
	)
}
