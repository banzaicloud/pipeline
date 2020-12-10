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

package semver

import (
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
)

func TestErrorInvalidVersion(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedErr        error
		inputVersionString string
	}{
		{
			caseDescription:    "invalid-version -> invalid-version error",
			expectedErr:        errors.New("invalid version invalid-version"),
			inputVersionString: "invalid-version",
		},
		{
			caseDescription:    "not-a-version -> not-a-version success",
			expectedErr:        errors.New("invalid version not-a-version"),
			inputVersionString: "not-a-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualErr := ErrorInvalidVersion(testCase.inputVersionString)

			require.EqualError(t, actualErr, testCase.expectedErr.Error())
		})
	}
}
