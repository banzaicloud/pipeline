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

	"github.com/stretchr/testify/require"
)

func TestCheckVersionString(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedErr     error
		inputCandidate  string
	}{
		{
			caseDescription: "valid -> true",
			expectedErr:     nil,
			inputCandidate:  "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "invalid -> false",
			expectedErr:     ErrorInvalidVersion("invalid-version"),
			inputCandidate:  "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualErr := CheckVersionString(testCase.inputCandidate)

			if testCase.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.expectedErr.Error())
			}
		})
	}
}

func TestIsValidVersionString(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedIsValid bool
		inputCandidate  string
	}{
		{
			caseDescription: "valid -> true",
			expectedIsValid: true,
			inputCandidate:  "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "invalid -> false",
			expectedIsValid: false,
			inputCandidate:  "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIsValid := IsValidVersionString(testCase.inputCandidate)

			require.Equal(t, testCase.expectedIsValid, actualIsValid)
		})
	}
}

func TestNewBuildVersion(t *testing.T) {
	type inputType struct {
		oldVersion       Version
		newBuildMetadata []string
	}

	testCases := []struct {
		caseDescription    string
		expectedNewVersion Version
		input              inputType
	}{
		{
			caseDescription:    "full -> success keep version, append build metadata",
			expectedNewVersion: "1.2.3-prerelease.4+build.5.1.metadata",
			input: inputType{
				oldVersion:       "1.2.3-prerelease.4+build.5",
				newBuildMetadata: []string{"1", "metadata"},
			},
		},
		{
			caseDescription:    "partial -> success keep version, append build metadata",
			expectedNewVersion: "6.7.8+1.metadata",
			input: inputType{
				oldVersion:       "6.7.8",
				newBuildMetadata: []string{"1", "metadata"},
			},
		},
		{
			caseDescription:    "invalid -> success return old version",
			expectedNewVersion: "invalid-version",
			input: inputType{
				oldVersion:       "invalid-version",
				newBuildMetadata: []string{"1", "metadata"},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewVersion := NewBuildVersion(testCase.input.oldVersion, testCase.input.newBuildMetadata...)

			require.Equal(t, testCase.expectedNewVersion, actualNewVersion)
		})
	}
}

func TestNewMajorVersion(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedNewVersion Version
		inputOldVersion    Version
	}{
		{
			caseDescription:    "full -> success increment major, 0 minor, 0 patch, nil prerelease, keep build",
			expectedNewVersion: "2.0.0+build.5",
			inputOldVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription:    "partial -> success increment minor, 0 patch",
			expectedNewVersion: "7.0.0",
			inputOldVersion:    "6.7.8",
		},
		{
			caseDescription:    "invalid -> success return old version",
			expectedNewVersion: "invalid-version",
			inputOldVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewVersion := NewMajorVersion(testCase.inputOldVersion)

			require.Equal(t, testCase.expectedNewVersion, actualNewVersion)
		})
	}
}

func TestNewMinorVersion(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedNewVersion Version
		inputOldVersion    Version
	}{
		{
			caseDescription:    "full -> success increment minor, 0 patch, nil prerelease, keep build",
			expectedNewVersion: "1.3.0+build.5",
			inputOldVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription:    "partial -> success increment minor, 0 patch",
			expectedNewVersion: "6.8.0",
			inputOldVersion:    "6.7.8",
		},
		{
			caseDescription:    "invalid -> success return old version",
			expectedNewVersion: "invalid-version",
			inputOldVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewVersion := NewMinorVersion(testCase.inputOldVersion)

			require.Equal(t, testCase.expectedNewVersion, actualNewVersion)
		})
	}
}

func TestNewPatchVersion(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedNewVersion Version
		inputOldVersion    Version
	}{
		{
			caseDescription:    "full -> success increment patch, nil prerelease",
			expectedNewVersion: "1.2.4+build.5",
			inputOldVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription:    "partial -> success increment patch",
			expectedNewVersion: "6.7.9",
			inputOldVersion:    "6.7.8",
		},
		{
			caseDescription:    "invalid -> success return old version",
			expectedNewVersion: "invalid-version",
			inputOldVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewVersion := NewPatchVersion(testCase.inputOldVersion)

			require.Equal(t, testCase.expectedNewVersion, actualNewVersion)
		})
	}
}

func TestNewPrereleaseVersion(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedNewVersion Version
		inputOldVersion    Version
	}{
		{
			caseDescription:    "no prerelease -> success append 1 prerelease",
			expectedNewVersion: "1.2.3-1",
			inputOldVersion:    "1.2.3",
		},
		{
			caseDescription:    "string last prerelease identifier -> success append 1 prerelease",
			expectedNewVersion: "1.2.3-prerelease.1+build.5",
			inputOldVersion:    "1.2.3-prerelease+build.5",
		},
		{
			caseDescription:    "decimal last prerelease identifier -> success increment prerelease",
			expectedNewVersion: "1.2.3-prerelease.2+build.5",
			inputOldVersion:    "1.2.3-prerelease.1+build.5",
		},
		{
			caseDescription:    "invalid -> success return old version",
			expectedNewVersion: "invalid-version",
			inputOldVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewVersion := NewPrereleaseVersion(testCase.inputOldVersion)

			require.Equal(t, testCase.expectedNewVersion, actualNewVersion)
		})
	}
}

func TestNewVersion(t *testing.T) {
	type inputType struct {
		major       int
		minor       int
		patch       int
		prereleases []string
		builds      []string
	}

	type outputType struct {
		expectedVersion Version
		expectedErr     error
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "full -> success",
			input: inputType{
				major:       1,
				minor:       2,
				patch:       3,
				prereleases: []string{"prerelease", "4"},
				builds:      []string{"build", "5"},
			},
			output: outputType{
				expectedVersion: Version("1.2.3-prerelease.4+build.5"),
				expectedErr:     nil,
			},
		},
		{
			caseDescription: "partial -> success",
			input: inputType{
				major: 6,
				minor: 7,
				patch: 8,
			},
			output: outputType{
				expectedVersion: Version("6.7.8"),
				expectedErr:     nil,
			},
		},
		{
			caseDescription: "invalid -> error",
			input: inputType{
				major:       0,
				minor:       0,
				patch:       1,
				prereleases: []string{"ű"},
			},
			output: outputType{
				expectedVersion: Version(""),
				expectedErr:     ErrorInvalidVersion("0.0.1-ű"),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualVersion, actualErr := NewVersion(
				testCase.input.major,
				testCase.input.minor,
				testCase.input.patch,
				testCase.input.prereleases,
				testCase.input.builds,
			)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedVersion, actualVersion)
		})
	}
}

func TestNewVersionOrPanic(t *testing.T) {
	type inputType struct {
		major       int
		minor       int
		patch       int
		prereleases []string
		builds      []string
	}

	type outputType struct {
		expectedVersion     Version
		expectedShouldPanic bool
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "full -> success",
			input: inputType{
				major:       1,
				minor:       2,
				patch:       3,
				prereleases: []string{"prerelease", "4"},
				builds:      []string{"build", "5"},
			},
			output: outputType{
				expectedVersion:     Version("1.2.3-prerelease.4+build.5"),
				expectedShouldPanic: false,
			},
		},
		{
			caseDescription: "partial -> success",
			input: inputType{
				major: 6,
				minor: 7,
				patch: 8,
			},
			output: outputType{
				expectedVersion:     Version("6.7.8"),
				expectedShouldPanic: false,
			},
		},
		{
			caseDescription: "invalid -> error",
			input: inputType{
				major:       0,
				minor:       0,
				patch:       1,
				prereleases: []string{"ű"},
			},
			output: outputType{
				expectedVersion:     Version(""),
				expectedShouldPanic: true,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			if testCase.output.expectedShouldPanic {
				require.Panics(t, func() {
					_ = NewVersionOrPanic(
						testCase.input.major,
						testCase.input.minor,
						testCase.input.patch,
						testCase.input.prereleases,
						testCase.input.builds,
					)
				})
			} else {
				actualVersion := NewVersionOrPanic(
					testCase.input.major,
					testCase.input.minor,
					testCase.input.patch,
					testCase.input.prereleases,
					testCase.input.builds,
				)

				require.Equal(t, testCase.output.expectedVersion, actualVersion)
			}
		})
	}
}

func TestNewVersionFromString(t *testing.T) {
	type outputType struct {
		expectedVersion Version
		expectedErr     error
	}

	testCases := []struct {
		caseDescription string
		inputCandidate  string
		output          outputType
	}{
		{
			caseDescription: "full -> success",
			inputCandidate:  "1.2.3-prerelease.4+build.5",
			output: outputType{
				expectedVersion: Version("1.2.3-prerelease.4+build.5"),
				expectedErr:     nil,
			},
		},
		{
			caseDescription: "partial -> success",
			inputCandidate:  "6.7.8",
			output: outputType{
				expectedVersion: Version("6.7.8"),
				expectedErr:     nil,
			},
		},
		{
			caseDescription: "invalid -> error",
			inputCandidate:  "invalid-version",
			output: outputType{
				expectedVersion: Version(""),
				expectedErr:     ErrorInvalidVersion("invalid-version"),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualVersion, actualErr := NewVersionFromString(testCase.inputCandidate)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedVersion, actualVersion)
		})
	}
}

func TestNewVersionFromStringOrPanic(t *testing.T) {
	type outputType struct {
		expectedVersion     Version
		expectedShouldPanic bool
	}

	testCases := []struct {
		caseDescription string
		inputCandidate  string
		output          outputType
	}{
		{
			caseDescription: "full -> success",
			inputCandidate:  "1.2.3-prerelease.4+build.5",
			output: outputType{
				expectedVersion:     Version("1.2.3-prerelease.4+build.5"),
				expectedShouldPanic: false,
			},
		},
		{
			caseDescription: "partial -> success",
			inputCandidate:  "6.7.8",
			output: outputType{
				expectedVersion:     Version("6.7.8"),
				expectedShouldPanic: false,
			},
		},
		{
			caseDescription: "invalid -> error",
			inputCandidate:  "invalid-version",
			output: outputType{
				expectedVersion:     Version(""),
				expectedShouldPanic: true,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			if testCase.output.expectedShouldPanic {
				require.Panics(t, func() { _ = NewVersionFromStringOrPanic(testCase.inputCandidate) })
			} else {
				actualVersion := NewVersionFromStringOrPanic(testCase.inputCandidate)

				require.Equal(t, testCase.output.expectedVersion, actualVersion)
			}
		})
	}
}

func TestVersionBuild(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedBuild   string
		inputVersion    Version
	}{
		{
			caseDescription: "full -> success",
			expectedBuild:   "build.5",
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> success",
			expectedBuild:   "",
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> success",
			expectedBuild:   "",
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualBuild := testCase.inputVersion.Build()

			require.Equal(t, testCase.expectedBuild, actualBuild)
		})
	}
}

func TestVersionBuilds(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedBuilds  []string
		inputVersion    Version
	}{
		{
			caseDescription: "full -> success",
			expectedBuilds:  []string{"build", "5"},
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> success",
			expectedBuilds:  nil,
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> success",
			expectedBuilds:  nil,
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualBuilds := testCase.inputVersion.Builds()

			require.Equal(t, testCase.expectedBuilds, actualBuilds)
		})
	}
}

func TestVersionCheck(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedErr     error
		inputVersion    Version
	}{
		{
			caseDescription: "full -> success",
			expectedErr:     nil,
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> success",
			expectedErr:     nil,
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> error",
			expectedErr:     ErrorInvalidVersion("invalid-version"),
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualErr := testCase.inputVersion.Check()

			if testCase.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.expectedErr.Error())
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	type inputType struct {
		version      Version
		otherVersion Version
	}

	testCases := []struct {
		caseDescription string
		expectedResult  Compared
		input           inputType
	}{
		{
			caseDescription: "major less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "2.0.0",
			},
		},
		{
			caseDescription: "major greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "2.0.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription: "major equal, minor less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "1.1.0",
			},
		},
		{
			caseDescription: "major equal, minor greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.1.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.2.0",
				otherVersion: "1.2.1",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.1",
				otherVersion: "1.2.0",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, one prerelease less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, one prerelease greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "1.2.3-prerelease",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, one prerelease decimal less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease.string",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, one prerelease decimal greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.3-prerelease.string",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, prereleases decimal less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease.1",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, prerelease decimal greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.3-prerelease.1",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, prerelease string less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.2.3-prerelease.alpha",
				otherVersion: "1.2.3-prerelease.beta",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, prerelease string greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.3-prerelease.beta",
				otherVersion: "1.2.3-prerelease.alpha",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, prerelease shorter less -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription: "major equal, minor equal, patch equal, prerelease longer greater -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease",
			},
		},
		{
			caseDescription: "all equal -> equal success",
			expectedResult:  ComparedEqual,
			input: inputType{
				version:      "1.2.3-prerelease.4",
				otherVersion: "1.2.3-prerelease.4",
			},
		},
		{
			caseDescription: "all equal with differing builds -> equal success",
			expectedResult:  ComparedEqual,
			input: inputType{
				version:      "1.2.3-prerelease.4+build.5",
				otherVersion: "1.2.3-prerelease.4+build.6",
			},
		},
		{
			caseDescription: "invalid version, valid zero other version -> equal success",
			expectedResult:  ComparedEqual,
			input: inputType{
				version:      "invalid-version",
				otherVersion: ZeroVersion,
			},
		},
		{
			caseDescription: "invalid version, valid non-zero other version -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "invalid-version",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription: "valid zero version, invalid other version -> equal success",
			expectedResult:  ComparedEqual,
			input: inputType{
				version:      ZeroVersion,
				otherVersion: "invalid-version",
			},
		},
		{
			caseDescription: "valid non-zero version, invalid other version -> greater success",
			expectedResult:  ComparedGreater,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "invalid-version",
			},
		},
		{
			caseDescription: "invalid version, invalid other version -> equal success",
			expectedResult:  ComparedEqual,
			input: inputType{
				version:      "invalid-version",
				otherVersion: "other-invalid-version",
			},
		},
		{
			caseDescription: "specification example 1 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "2.0.0",
			},
		},
		{
			caseDescription: "specification example 2 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "2.0.0",
				otherVersion: "2.1.0",
			},
		},
		{
			caseDescription: "specification example 3 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "2.1.0",
				otherVersion: "2.1.1",
			},
		},
		{
			caseDescription: "specification example 4 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-alpha",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription: "specification example 5 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-alpha",
				otherVersion: "1.0.0-alpha.1",
			},
		},
		{
			caseDescription: "specification example 6 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-alpha.1",
				otherVersion: "1.0.0-alpha.beta",
			},
		},
		{
			caseDescription: "specification example 7 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription: "specification example 8 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription: "specification example 9 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription: "specification example 10 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-beta",
				otherVersion: "1.0.0-beta.2",
			},
		},
		{
			caseDescription: "specification example 11 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-beta.2",
				otherVersion: "1.0.0-beta.11",
			},
		},
		{
			caseDescription: "specification example 12 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-beta.11",
				otherVersion: "1.0.0-rc.1",
			},
		},
		{
			caseDescription: "specification example 13 -> less success",
			expectedResult:  ComparedLess,
			input: inputType{
				version:      "1.0.0-rc.1",
				otherVersion: "1.0.0",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualResult := testCase.input.version.Compare(testCase.input.otherVersion)

			require.Equal(t, testCase.expectedResult, actualResult)
		})
	}
}

func TestVersionEquals(t *testing.T) {
	type inputType struct {
		version      Version
		otherVersion Version
	}

	testCases := []struct {
		caseDescription  string
		expectedAreEqual bool
		input            inputType
	}{
		{
			caseDescription:  "major difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "0.0.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:  "minor difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "1.1.0",
				otherVersion: "1.2.0",
			},
		},
		{
			caseDescription:  "patch difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "1.2.2",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription:  "version prerelease difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "2.2.3",
			},
		},
		{
			caseDescription:  "other prerelease difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "2.2.3-prerelease",
			},
		},
		{
			caseDescription:  "decimal prerelease difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "1.2.3-prerelease.3",
				otherVersion: "2.2.3-prerelease.4",
			},
		},
		{
			caseDescription:  "string prerelease difference -> false success",
			expectedAreEqual: false,
			input: inputType{
				version:      "1.2.3-prerelease.4.alpha",
				otherVersion: "2.2.3-prerelease.4.beta",
			},
		},
		{
			caseDescription:  "no build equal -> true success",
			expectedAreEqual: true,
			input: inputType{
				version:      "1.2.3-prerelease.4",
				otherVersion: "1.2.3-prerelease.4",
			},
		},
		{
			caseDescription:  "build difference equal -> true success",
			expectedAreEqual: true,
			input: inputType{
				version:      "1.2.3-prerelease.4+build.4",
				otherVersion: "1.2.3-prerelease.4+build.5",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualAreEqual := testCase.input.version.Equals(testCase.input.otherVersion)

			require.Equal(t, testCase.expectedAreEqual, actualAreEqual)
		})
	}
}

func TestVersionIsGreaterThan(t *testing.T) {
	type inputType struct {
		version      Version
		otherVersion Version
	}

	testCases := []struct {
		caseDescription       string
		expectedIsGreaterThan bool
		input                 inputType
	}{
		{
			caseDescription:       "major less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "2.0.0",
			},
		},
		{
			caseDescription:       "major greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "2.0.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:       "major equal, minor less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "1.1.0",
			},
		},
		{
			caseDescription:       "major equal, minor greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.1.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.0",
				otherVersion: "1.2.1",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.1",
				otherVersion: "1.2.0",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, one prerelease less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, one prerelease greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "1.2.3-prerelease",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, one prerelease decimal less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease.string",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, one prerelease decimal greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.string",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, prereleases decimal less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease.1",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, prerelease decimal greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.1",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, prerelease string less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.alpha",
				otherVersion: "1.2.3-prerelease.beta",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, prerelease string greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.beta",
				otherVersion: "1.2.3-prerelease.alpha",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, prerelease shorter less -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription:       "major equal, minor equal, patch equal, prerelease longer greater -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease",
			},
		},
		{
			caseDescription:       "all equal -> equal success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.4",
				otherVersion: "1.2.3-prerelease.4",
			},
		},
		{
			caseDescription:       "all equal with differing builds -> equal success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.4+build.5",
				otherVersion: "1.2.3-prerelease.4+build.6",
			},
		},
		{
			caseDescription:       "invalid version, valid zero other version -> equal success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "invalid-version",
				otherVersion: ZeroVersion,
			},
		},
		{
			caseDescription:       "invalid version, valid non-zero other version -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "invalid-version",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription:       "valid zero version, invalid other version -> equal success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      ZeroVersion,
				otherVersion: "invalid-version",
			},
		},
		{
			caseDescription:       "valid non-zero version, invalid other version -> greater success",
			expectedIsGreaterThan: true,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "invalid-version",
			},
		},
		{
			caseDescription:       "invalid version, invalid other version -> equal success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "invalid-version",
				otherVersion: "other-invalid-version",
			},
		},
		{
			caseDescription:       "specification example 1 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "2.0.0",
			},
		},
		{
			caseDescription:       "specification example 2 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "2.0.0",
				otherVersion: "2.1.0",
			},
		},
		{
			caseDescription:       "specification example 3 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "2.1.0",
				otherVersion: "2.1.1",
			},
		},
		{
			caseDescription:       "specification example 4 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-alpha",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:       "specification example 5 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-alpha",
				otherVersion: "1.0.0-alpha.1",
			},
		},
		{
			caseDescription:       "specification example 6 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-alpha.1",
				otherVersion: "1.0.0-alpha.beta",
			},
		},
		{
			caseDescription:       "specification example 7 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription:       "specification example 8 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription:       "specification example 9 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription:       "specification example 10 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-beta",
				otherVersion: "1.0.0-beta.2",
			},
		},
		{
			caseDescription:       "specification example 11 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-beta.2",
				otherVersion: "1.0.0-beta.11",
			},
		},
		{
			caseDescription:       "specification example 12 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-beta.11",
				otherVersion: "1.0.0-rc.1",
			},
		},
		{
			caseDescription:       "specification example 13 -> less success",
			expectedIsGreaterThan: false,
			input: inputType{
				version:      "1.0.0-rc.1",
				otherVersion: "1.0.0",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIsGreaterThan := testCase.input.version.IsGreaterThan(testCase.input.otherVersion)

			require.Equal(t, testCase.expectedIsGreaterThan, actualIsGreaterThan)
		})
	}
}

func TestVersionIsInRange(t *testing.T) {
	type inputType struct {
		version                Version
		inclusiveLowerBoundary Version
		exclusiveUpperBoundary Version
	}

	testCases := []struct {
		caseDescription   string
		expectedIsInRange bool
		input             inputType
	}{
		{
			caseDescription:   "major below lower boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "0.0.0",
				inclusiveLowerBoundary: "1.0.0",
				exclusiveUpperBoundary: "3.0.0",
			},
		},
		{
			caseDescription:   "major equals lower boundary -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.0.0",
				inclusiveLowerBoundary: "1.0.0",
				exclusiveUpperBoundary: "3.0.0",
			},
		},
		{
			caseDescription:   "major between boundaries -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "2.0.0",
				inclusiveLowerBoundary: "1.0.0",
				exclusiveUpperBoundary: "3.0.0",
			},
		},
		{
			caseDescription:   "major equals upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "3.0.0",
				inclusiveLowerBoundary: "1.0.0",
				exclusiveUpperBoundary: "3.0.0",
			},
		},
		{
			caseDescription:   "major above upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "4.0.0",
				inclusiveLowerBoundary: "1.0.0",
				exclusiveUpperBoundary: "3.0.0",
			},
		},
		{
			caseDescription:   "minor below lower boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.1.0",
				inclusiveLowerBoundary: "1.2.0",
				exclusiveUpperBoundary: "1.4.0",
			},
		},
		{
			caseDescription:   "minor equals lower boundary -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.0",
				inclusiveLowerBoundary: "1.2.0",
				exclusiveUpperBoundary: "1.4.0",
			},
		},
		{
			caseDescription:   "minor between boundaries -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.3.0",
				inclusiveLowerBoundary: "1.2.0",
				exclusiveUpperBoundary: "1.4.0",
			},
		},
		{
			caseDescription:   "minor equals upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.4.0",
				inclusiveLowerBoundary: "1.2.0",
				exclusiveUpperBoundary: "1.4.0",
			},
		},
		{
			caseDescription:   "minor above upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.5.0",
				inclusiveLowerBoundary: "1.2.0",
				exclusiveUpperBoundary: "1.4.0",
			},
		},
		{
			caseDescription:   "patch below lower boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.2",
				inclusiveLowerBoundary: "1.2.3",
				exclusiveUpperBoundary: "1.2.5",
			},
		},
		{
			caseDescription:   "patch equals lower boundary -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3",
				inclusiveLowerBoundary: "1.2.3",
				exclusiveUpperBoundary: "1.2.5",
			},
		},
		{
			caseDescription:   "patch between boundaries -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.4",
				inclusiveLowerBoundary: "1.2.3",
				exclusiveUpperBoundary: "1.2.5",
			},
		},
		{
			caseDescription:   "patch equals upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.5",
				inclusiveLowerBoundary: "1.2.3",
				exclusiveUpperBoundary: "1.2.5",
			},
		},
		{
			caseDescription:   "patch above upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.6",
				inclusiveLowerBoundary: "1.2.3",
				exclusiveUpperBoundary: "1.2.5",
			},
		},
		{
			caseDescription:   "prerelease below lower boundary by prerelease -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.3-prerelease",
				inclusiveLowerBoundary: "1.2.3",
				exclusiveUpperBoundary: "1.2.5",
			},
		},
		{
			caseDescription:   "prerelease below lower boundary by prerelease identifier -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.3-prerelease.3",
				inclusiveLowerBoundary: "1.2.3-prerelease.4",
				exclusiveUpperBoundary: "1.2.3-prerelease.6",
			},
		},
		{
			caseDescription:   "prerelease equals lower boundary -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.4",
				inclusiveLowerBoundary: "1.2.3-prerelease.4",
				exclusiveUpperBoundary: "1.2.3-prerelease.6",
			},
		},
		{
			caseDescription:   "prerelease between boundaries -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.5",
				inclusiveLowerBoundary: "1.2.3-prerelease.4",
				exclusiveUpperBoundary: "1.2.3-prerelease.6",
			},
		},
		{
			caseDescription:   "prerelease equals upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.3-prerelease.6",
				inclusiveLowerBoundary: "1.2.3-prerelease.4",
				exclusiveUpperBoundary: "1.2.3-prerelease.6",
			},
		},
		{
			caseDescription:   "prerelease above upper boundary -> false success",
			expectedIsInRange: false,
			input: inputType{
				version:                "1.2.3-prerelease.7",
				inclusiveLowerBoundary: "1.2.3-prerelease.4",
				exclusiveUpperBoundary: "1.2.3-prerelease.6",
			},
		},
		{
			caseDescription:   "build does not affect range, below lower boundary -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.4+build.4",
				inclusiveLowerBoundary: "1.2.3-prerelease.4+build.5",
				exclusiveUpperBoundary: "1.2.3-prerelease.6+build.7",
			},
		},
		{
			caseDescription:   "build does not affect range, equals lower boundary -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.4+build.5",
				inclusiveLowerBoundary: "1.2.3-prerelease.4+build.5",
				exclusiveUpperBoundary: "1.2.3-prerelease.6+build.7",
			},
		},
		{
			caseDescription:   "build does not affect range, between boundaries -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.4+build.6",
				inclusiveLowerBoundary: "1.2.3-prerelease.4+build.5",
				exclusiveUpperBoundary: "1.2.3-prerelease.6+build.7",
			},
		},
		{
			caseDescription:   "build does not affect range, equals upper boundary  -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.4+build.7",
				inclusiveLowerBoundary: "1.2.3-prerelease.4+build.5",
				exclusiveUpperBoundary: "1.2.3-prerelease.6+build.7",
			},
		},
		{
			caseDescription:   "build does not affect range, above upper boundary  -> true success",
			expectedIsInRange: true,
			input: inputType{
				version:                "1.2.3-prerelease.4+build.8",
				inclusiveLowerBoundary: "1.2.3-prerelease.4+build.5",
				exclusiveUpperBoundary: "1.2.3-prerelease.6+build.7",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIsInRange := testCase.input.version.IsInRange(
				testCase.input.inclusiveLowerBoundary,
				testCase.input.exclusiveUpperBoundary,
			)

			require.Equal(t, testCase.expectedIsInRange, actualIsInRange)
		})
	}
}

func TestVersionIsLessThan(t *testing.T) {
	type inputType struct {
		version      Version
		otherVersion Version
	}

	testCases := []struct {
		caseDescription    string
		expectedIsLessThan bool
		input              inputType
	}{
		{
			caseDescription:    "major less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "2.0.0",
			},
		},
		{
			caseDescription:    "major greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "2.0.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:    "major equal, minor less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "1.1.0",
			},
		},
		{
			caseDescription:    "major equal, minor greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.1.0",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.2.0",
				otherVersion: "1.2.1",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.1",
				otherVersion: "1.2.0",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, one prerelease less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, one prerelease greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "1.2.3-prerelease",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, one prerelease decimal less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease.string",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, one prerelease decimal greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.string",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, prereleases decimal less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease.1",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, prerelease decimal greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.1",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, prerelease string less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.2.3-prerelease.alpha",
				otherVersion: "1.2.3-prerelease.beta",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, prerelease string greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.beta",
				otherVersion: "1.2.3-prerelease.alpha",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, prerelease shorter less -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.2.3-prerelease",
				otherVersion: "1.2.3-prerelease.0",
			},
		},
		{
			caseDescription:    "major equal, minor equal, patch equal, prerelease longer greater -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.0",
				otherVersion: "1.2.3-prerelease",
			},
		},
		{
			caseDescription:    "all equal -> equal success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.4",
				otherVersion: "1.2.3-prerelease.4",
			},
		},
		{
			caseDescription:    "all equal with differing builds -> equal success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3-prerelease.4+build.5",
				otherVersion: "1.2.3-prerelease.4+build.6",
			},
		},
		{
			caseDescription:    "invalid version, valid zero other version -> equal success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "invalid-version",
				otherVersion: ZeroVersion,
			},
		},
		{
			caseDescription:    "invalid version, valid non-zero other version -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "invalid-version",
				otherVersion: "1.2.3",
			},
		},
		{
			caseDescription:    "valid zero version, invalid other version -> equal success",
			expectedIsLessThan: false,
			input: inputType{
				version:      ZeroVersion,
				otherVersion: "invalid-version",
			},
		},
		{
			caseDescription:    "valid non-zero version, invalid other version -> greater success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "1.2.3",
				otherVersion: "invalid-version",
			},
		},
		{
			caseDescription:    "invalid version, invalid other version -> equal success",
			expectedIsLessThan: false,
			input: inputType{
				version:      "invalid-version",
				otherVersion: "other-invalid-version",
			},
		},
		{
			caseDescription:    "specification example 1 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0",
				otherVersion: "2.0.0",
			},
		},
		{
			caseDescription:    "specification example 2 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "2.0.0",
				otherVersion: "2.1.0",
			},
		},
		{
			caseDescription:    "specification example 3 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "2.1.0",
				otherVersion: "2.1.1",
			},
		},
		{
			caseDescription:    "specification example 4 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-alpha",
				otherVersion: "1.0.0",
			},
		},
		{
			caseDescription:    "specification example 5 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-alpha",
				otherVersion: "1.0.0-alpha.1",
			},
		},
		{
			caseDescription:    "specification example 6 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-alpha.1",
				otherVersion: "1.0.0-alpha.beta",
			},
		},
		{
			caseDescription:    "specification example 7 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription:    "specification example 8 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription:    "specification example 9 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-alpha.beta",
				otherVersion: "1.0.0-beta",
			},
		},
		{
			caseDescription:    "specification example 10 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-beta",
				otherVersion: "1.0.0-beta.2",
			},
		},
		{
			caseDescription:    "specification example 11 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-beta.2",
				otherVersion: "1.0.0-beta.11",
			},
		},
		{
			caseDescription:    "specification example 12 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-beta.11",
				otherVersion: "1.0.0-rc.1",
			},
		},
		{
			caseDescription:    "specification example 13 -> less success",
			expectedIsLessThan: true,
			input: inputType{
				version:      "1.0.0-rc.1",
				otherVersion: "1.0.0",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIsLessThan := testCase.input.version.IsLessThan(testCase.input.otherVersion)

			require.Equal(t, testCase.expectedIsLessThan, actualIsLessThan)
		})
	}
}

func TestVersionIsValid(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedIsValid bool
		inputVersion    Version
	}{
		{
			caseDescription: "full -> valid",
			expectedIsValid: true,
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> valid",
			expectedIsValid: true,
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> invalid",
			expectedIsValid: false,
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualIsValid := testCase.inputVersion.IsValid()

			require.Equal(t, testCase.expectedIsValid, actualIsValid)
		})
	}
}

func TestVersionMajor(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedMajor   int
		inputVersion    Version
	}{
		{
			caseDescription: "full -> success",
			expectedMajor:   1,
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> success",
			expectedMajor:   6,
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> success",
			expectedMajor:   0,
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualMajor := testCase.inputVersion.Major()

			require.Equal(t, testCase.expectedMajor, actualMajor)
		})
	}
}

func TestVersionMinor(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedMinor   int
		inputVersion    Version
	}{
		{
			caseDescription: "full -> success",
			expectedMinor:   2,
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> success",
			expectedMinor:   7,
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> success",
			expectedMinor:   0,
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualMinor := testCase.inputVersion.Minor()

			require.Equal(t, testCase.expectedMinor, actualMinor)
		})
	}
}

func TestVersionParts(t *testing.T) {
	type outputType struct {
		expectedMajor       int
		expectedMinor       int
		expectedPatch       int
		expectedPrereleases []string
		expectedBuilds      []string
	}

	testCases := []struct {
		caseDescription string
		inputVersion    Version
		output          outputType
	}{
		{
			caseDescription: "full -> success",
			inputVersion:    "1.2.3-prerelease.4+build.5",
			output: outputType{
				expectedMajor:       1,
				expectedMinor:       2,
				expectedPatch:       3,
				expectedPrereleases: []string{"prerelease", "4"},
				expectedBuilds:      []string{"build", "5"},
			},
		},
		{
			caseDescription: "partial -> success",
			inputVersion:    "6.7.8",
			output: outputType{
				expectedMajor:       6,
				expectedMinor:       7,
				expectedPatch:       8,
				expectedPrereleases: nil,
				expectedBuilds:      nil,
			},
		},
		{
			caseDescription: "invalid -> success",
			inputVersion:    "invalid-version",
			output: outputType{
				expectedMajor:       0,
				expectedMinor:       0,
				expectedPatch:       0,
				expectedPrereleases: nil,
				expectedBuilds:      nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualMajor, actualMinor, actualPatch, actualPrereleases, actualBuilds := testCase.inputVersion.Parts()

			require.Equal(t, testCase.output.expectedMajor, actualMajor)
			require.Equal(t, testCase.output.expectedMinor, actualMinor)
			require.Equal(t, testCase.output.expectedPatch, actualPatch)
			require.Equal(t, testCase.output.expectedPrereleases, actualPrereleases)
			require.Equal(t, testCase.output.expectedBuilds, actualBuilds)
		})
	}
}

func TestVersionPatch(t *testing.T) {
	testCases := []struct {
		caseDescription string
		expectedPatch   int
		inputVersion    Version
	}{
		{
			caseDescription: "full -> success",
			expectedPatch:   3,
			inputVersion:    "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription: "partial -> success",
			expectedPatch:   8,
			inputVersion:    "6.7.8",
		},
		{
			caseDescription: "invalid -> success",
			expectedPatch:   0,
			inputVersion:    "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualPatch := testCase.inputVersion.Patch()

			require.Equal(t, testCase.expectedPatch, actualPatch)
		})
	}
}

func TestVersionPrerelease(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedPrerelease string
		inputVersion       Version
	}{
		{
			caseDescription:    "full -> success",
			expectedPrerelease: "prerelease.4",
			inputVersion:       "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription:    "partial -> success",
			expectedPrerelease: "",
			inputVersion:       "6.7.8",
		},
		{
			caseDescription:    "invalid -> success",
			expectedPrerelease: "",
			inputVersion:       "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualPrerelease := testCase.inputVersion.Prerelease()

			require.Equal(t, testCase.expectedPrerelease, actualPrerelease)
		})
	}
}

func TestVersionPrereleases(t *testing.T) {
	testCases := []struct {
		caseDescription     string
		expectedPrereleases []string
		inputVersion        Version
	}{
		{
			caseDescription:     "full -> success",
			expectedPrereleases: []string{"prerelease", "4"},
			inputVersion:        "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription:     "partial -> success",
			expectedPrereleases: nil,
			inputVersion:        "6.7.8",
		},
		{
			caseDescription:     "invalid -> success",
			expectedPrereleases: nil,
			inputVersion:        "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualPrereleases := testCase.inputVersion.Prereleases()

			require.Equal(t, testCase.expectedPrereleases, actualPrereleases)
		})
	}
}

func TestVersionRawParts(t *testing.T) {
	type outputType struct {
		expectedMajor      string
		expectedMinor      string
		expectedPatch      string
		expectedPrerelease string
		expectedBuild      string
	}

	testCases := []struct {
		caseDescription string
		inputVersion    Version
		output          outputType
	}{
		{
			caseDescription: "full -> success",
			inputVersion:    "1.2.3-prerelease.4+build.5",
			output: outputType{
				expectedMajor:      "1",
				expectedMinor:      "2",
				expectedPatch:      "3",
				expectedPrerelease: "prerelease.4",
				expectedBuild:      "build.5",
			},
		},
		{
			caseDescription: "partial -> success",
			inputVersion:    "6.7.8",
			output: outputType{
				expectedMajor:      "6",
				expectedMinor:      "7",
				expectedPatch:      "8",
				expectedPrerelease: "",
				expectedBuild:      "",
			},
		},
		{
			caseDescription: "invalid -> success",
			inputVersion:    "invalid-version",
			output: outputType{
				expectedMajor:      "0",
				expectedMinor:      "0",
				expectedPatch:      "0",
				expectedPrerelease: "",
				expectedBuild:      "",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualMajor, actualMinor, actualPatch, actualPrerelease, actualBuild := testCase.inputVersion.RawParts()

			require.Equal(t, testCase.output.expectedMajor, actualMajor)
			require.Equal(t, testCase.output.expectedMinor, actualMinor)
			require.Equal(t, testCase.output.expectedPatch, actualPatch)
			require.Equal(t, testCase.output.expectedPrerelease, actualPrerelease)
			require.Equal(t, testCase.output.expectedBuild, actualBuild)
		})
	}
}

func TestVersionString(t *testing.T) {
	testCases := []struct {
		caseDescription       string
		expectedVersionString string
		inputVersion          Version
	}{
		{
			caseDescription:       "1.2.3-prerelease.4+build.5 -> success",
			expectedVersionString: "1.2.3-prerelease.4+build.5",
			inputVersion:          "1.2.3-prerelease.4+build.5",
		},
		{
			caseDescription:       "invalid -> success",
			expectedVersionString: "invalid-version",
			inputVersion:          "invalid-version",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualVersionString := testCase.inputVersion.String()

			require.Equal(t, testCase.expectedVersionString, actualVersionString)
		})
	}
}
