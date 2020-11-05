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

package eksworkflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/testsuite"
)

func TestCalculateNodePoolVersionActivity(t *testing.T) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()

	NewCalculateNodePoolVersionActivity().Register(env)

	input := CalculateNodePoolVersionActivityInput{
		Image:      "ami-xxxxxxxxxxxxx",
		VolumeSize: 50,
	}

	v, err := env.ExecuteActivity(CalculateNodePoolVersionActivityName, input)
	require.NoError(t, err)

	var output CalculateNodePoolVersionActivityOutput

	err = v.Get(&output)
	require.NoError(t, err)

	assert.Equal(
		t,
		CalculateNodePoolVersionActivityOutput{
			Version: "ded58f7cde1bcf3a39227af824dab18086487c4c",
		},
		output,
	)
}
