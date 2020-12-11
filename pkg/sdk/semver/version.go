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

// Version provides an implicitly string-serializable semantic version type
// implementation with semantic version helper functions.
//
// The package implements the Semantic Versioning 2.0.0 specification from
// https://semver.org/.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// BuildPartRawRegex contains the raw regular expression string decsribing
	// the allowed string schema of a build part of a semantic version.
	BuildPartRawRegex = `[0-9a-zA-Z-]+`

	// DecimalDigitsRawRegex contains the raw regular expression string
	// describing the allowed string schema of a string containing only decimal
	// digits.
	DecimalDigitsRawRegex = `[0-9]+`

	// PrereleasePartRawRegex contains the raw regular expression string
	// describing the allowed string schema of a prerelease part of a semantic
	// version.
	PrereleasePartRawRegex = VersionPartRawRegex + `|\d*[a-zA-Z-][0-9a-zA-Z-]*`

	// VersionRawRegex contains the raw regular expression string describing the
	// allowed string schema of a semantic version.
	VersionRawRegex = `^(` + VersionPartRawRegex + `)\.(` + VersionPartRawRegex + `)\.(` + VersionPartRawRegex + `)` +
		`(?:-((?:` + PrereleasePartRawRegex + `)(?:\.(?:` + PrereleasePartRawRegex + `))*))?` +
		`(?:\+(` + BuildPartRawRegex + `(?:\.` + BuildPartRawRegex + `)*))?$`

	// VersionPartRawRegex contains the raw regular expression string describing
	// the allowed string schema of a major, minor or patch part of a semantic
	// version.
	VersionPartRawRegex = `0|[1-9]\d*`

	// ZeroVersion is the version representation of the "0.0.0" version.
	ZeroVersion = Version("0.0.0")
)

var (
	// DecimalDigitsRegex is the regular expression describing the allowed string
	// schema of a string containing only decimal digits.
	DecimalDigitsRegex = regexp.MustCompile(DecimalDigitsRawRegex) // nolint:gochecknoglobals // Note: intentional.

	// VersionRegex is the regular expression describing the allowed string schema
	// of a semantic version.
	VersionRegex = regexp.MustCompile(VersionRawRegex) // nolint:gochecknoglobals // Note: intentional.
)

// Version represents a semantic version with corresponding helper functions for
// structured part access and is aiming to be a string drop-in replacement type
// (with the exception of operators for which helper functions are provided).
//
// WARNING: invalid semantic versions are treated similarly to the "0.0.0"
// version.
type Version string

// NewBuildVersion creates a version from the specified value by appending the
// provided build metadata to the possibly existing values.
//
// WARNING: f the old version is invalid, it is returned without any changes.
func NewBuildVersion(oldVersion Version, newBuildMetadata ...string) (newVersion Version) {
	if !oldVersion.IsValid() {
		return oldVersion
	}

	oldMajor, oldMinor, oldPatch, oldPrereleases, oldBuilds := oldVersion.Parts()

	// Note: validation ensures no error.
	newVersion, _ = NewVersion(oldMajor, oldMinor, oldPatch, oldPrereleases, append(oldBuilds, newBuildMetadata...))

	return newVersion
}

// NewMajorVersion creates a version from the specified value by incrementing
// the major version.
//
// WARNING: f the old version is invalid, it is returned without any changes.
func NewMajorVersion(oldVersion Version) (newVersion Version) {
	if !oldVersion.IsValid() {
		return oldVersion
	}

	oldMajor, _, _, _, oldBuilds := oldVersion.Parts() // nolint:dogsled // Note: intentional performance/usability.

	// Note: validation ensures no error.
	newVersion, _ = NewVersion(oldMajor+1, 0, 0, nil, oldBuilds)

	return newVersion
}

// NewMinorVersion creates a version from the specified value by incrementing
// the minor version.
//
// WARNING: f the old version is invalid, it is returned without any changes.
func NewMinorVersion(oldVersion Version) (newVersion Version) {
	if !oldVersion.IsValid() {
		return oldVersion
	}

	oldMajor, oldMinor, _, _, oldBuilds := oldVersion.Parts()

	// Note: validation ensures no error.
	newVersion, _ = NewVersion(oldMajor, oldMinor+1, 0, nil, oldBuilds)

	return newVersion
}

// NewPatchVersion creates a version from the specified value by incrementing
// the patch version.
//
// WARNING: f the old version is invalid, it is returned without any changes.
func NewPatchVersion(oldVersion Version) (newVersion Version) {
	if !oldVersion.IsValid() {
		return oldVersion
	}

	oldMajor, oldMinor, oldPatch, _, oldBuilds := oldVersion.Parts()

	// Note: validation ensures no error.
	newVersion, _ = NewVersion(oldMajor, oldMinor, oldPatch+1, nil, oldBuilds)

	return newVersion
}

// NewPrereleaseVersion creates a version from the specified value by doing the
// first applicable action of the following list in order to increment the
// prerelease version>
//
// 1. IF there is no prerelease version, it is added with a decimal 1
// identifier.
//
// 2. If there is a prerelease version already and the last identifier of the
// prerelease version is not a decimal, a new prerelease identifier is added
// with a decimal 1 value.
//
// 3. If there is a prerelease version already and the last identifier of the
// prerelease version is a decimal, it is incremented by 1.
//
// WARNING: if the old version is invalid, it is returned without any changes.
func NewPrereleaseVersion(oldVersion Version) (newVersion Version) {
	if !oldVersion.IsValid() {
		return oldVersion
	}

	oldMajor, oldMinor, oldPatch, oldPrereleases, oldBuilds := oldVersion.Parts()

	newPrereleases := make([]string, len(oldPrereleases))
	_ = copy(newPrereleases, oldPrereleases) // Note: creation ensures the correct capacity.

	if len(oldPrereleases) == 0 ||
		!DecimalDigitsRegex.MatchString(oldPrereleases[len(oldPrereleases)-1]) {
		newPrereleases = append(newPrereleases, "1")
	} else {
		oldPreleaseDecimal, _ := strconv.Atoi(oldPrereleases[len(oldPrereleases)-1]) // Note: regex ensures no error.

		newPrereleases[len(newPrereleases)-1] = strconv.FormatInt(int64(oldPreleaseDecimal+1), 10)
	}

	// Note: validation ensures no error.
	newVersion, _ = NewVersion(oldMajor, oldMinor, oldPatch, newPrereleases, oldBuilds)

	return newVersion
}

// NewVersion returns a version object created from the specified parts or an
// error.
func NewVersion(major, minor, patch int, prereleases, builds []string) (version Version, err error) {
	versionString := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	if len(prereleases) > 0 {
		versionString += "-" + strings.Join(prereleases, ".")
	}

	if len(builds) > 0 {
		versionString += "+" + strings.Join(builds, ".")
	}

	return NewVersionFromString(versionString)
}

// NewVersionFromString returns a version object created from the specified
// string or an error.
func NewVersionFromString(candidate string) (version Version, err error) {
	if !IsValidVersionString(candidate) {
		return Version(""), ErrorInvalidVersion(candidate)
	}

	return Version(candidate), nil
}

// NewVersionFromStringOrPanic returns a version object created from the
// specified string or panics.
func NewVersionFromStringOrPanic(candidate string) (version Version) {
	version, err := NewVersionFromString(candidate)
	if err != nil {
		panic(err)
	}

	return version
}

// NewVersion returns a version object created from the specified parts or
// panics.
func NewVersionOrPanic(major, minor, patch int, prereleases, builds []string) (version Version) {
	version, err := NewVersion(major, minor, patch, prereleases, builds)
	if err != nil {
		panic(err)
	}

	return version
}

// Build returns the build part of a semantic version as a string.
//
// WARNING: for an invalid semantic version this function returns the empty
// string.
//
// WARNING: the returned build part does not contain the leading plus.
func (version Version) Build() (build string) {
	_, _, _, _, build = version.RawParts() // nolint:dogsled // Note: intentional performance/usability choice.

	return build
}

// Builds returns the build parts of a semantic version as a string slice.
//
// WARNING: for an invalid semantic version this function returns nil.
//
// WARNING: the returned build parts do not contain the leading plus.
func (version Version) Builds() (builds []string) {
	_, _, _, _, builds = version.Parts() // nolint:dogsled // Note: intentional performance/usability choice.

	return builds
}

// Check checks whether the version is a valid semantic version and returns
// error if it is not.
func (version Version) Check() (err error) {
	if !version.IsValid() {
		return ErrorInvalidVersion(version.String())
	}

	return nil
}

// Compare compares the receiver version to the specified other version
// comparing major, minor, patch, prerelease version parts.
//
// WARNING: the comparison ignores the build metadata part of the versions.
//
// WARNING: invalid semantic versions are treated similarly to the "0.0.0"
// version.
func (version Version) Compare(otherVersion Version) (result Compared) {
	versionMajor, versionMinor, versionPatch, versionPrereleases, _ := version.Parts()
	otherMajor, otherMinor, otherPatch, otherPrereleases, _ := otherVersion.Parts()

	if versionMajor != otherMajor {
		return CompareInts(versionMajor, otherMajor)
	} else if versionMinor != otherMinor {
		return CompareInts(versionMinor, otherMinor)
	} else if versionPatch != otherPatch {
		return CompareInts(versionPatch, otherPatch)
	}

	versionPrereleaseCount := len(versionPrereleases)
	otherVersionPrereleaseCount := len(otherPrereleases)
	if (versionPrereleaseCount == 0) != (otherVersionPrereleaseCount == 0) {
		// Note: exactly one of the counts is 0, the version with 0 prerelease
		// count is the greater version of the two (reversing the comparison).
		return CompareInts(otherVersionPrereleaseCount, versionPrereleaseCount)
	}

	for index := 0; index != versionPrereleaseCount && index != otherVersionPrereleaseCount; index++ {
		versionPart := versionPrereleases[index]
		otherPart := otherPrereleases[index]
		if versionPart == otherPart { // Note: parts are alphabetically equal, they would be decimally equal as well.
			continue
		}

		isVersionPartDecimal := DecimalDigitsRegex.MatchString(versionPart)
		isOtherPartDecimal := DecimalDigitsRegex.MatchString(otherPart)
		if isVersionPartDecimal != isOtherPartDecimal &&
			isVersionPartDecimal { // Note: decimal is less than non-decimal.
			return ComparedLess
		} else if isVersionPartDecimal != isOtherPartDecimal &&
			isOtherPartDecimal {
			return ComparedGreater
		} else if isVersionPartDecimal { // && isOtherPartDecimal // Note: compare decimally.
			versionDecimalPart, _ := strconv.Atoi(versionPart) // Note: the regex and condition ensures no error here.
			otherDecimalPart, _ := strconv.Atoi(otherPart)     // Note: the regex and condition ensures no error here.

			result = CompareInts(versionDecimalPart, otherDecimalPart)
			if result != ComparedEqual {
				return result
			}
		} else { // Note: compare alphabetically.
			result = CompareStrings(versionPart, otherPart)
			if result != ComparedEqual {
				return result
			}
		}
	}

	return CompareInts(versionPrereleaseCount, otherVersionPrereleaseCount)
}

// Equals returns true if the receiver version is equal to the specified
// version, false otherwise.
//
// WARNING: the comparison ignores the build metadata part of the versions.
//
// WARNING: invalid semantic versions are treated similarly to the "0.0.0"
// version.
func (version Version) Equals(otherVersion Version) (areEqual bool) {
	return version.Compare(otherVersion) == ComparedEqual
}

// IsGreaterThan returns true if the receiver version is greater than the
// specified version comparing major, minor, patch and prerelease versions
// according to the semantic version specification.
//
// WARNING: the comparison ignores the build metadata part of the versions.
//
// WARNING: invalid semantic versions are treated similarly to the "0.0.0"
// version.
func (version Version) IsGreaterThan(otherVersion Version) (isGreaterThan bool) {
	return version.Compare(otherVersion) == ComparedGreater
}

// IsInRange returns true if the receiver version is in the range of the
// specified boundaries and false otherwise.
//
// WARNING: the lower boundary is inclusive, while the upper boundary is
// exclusive. This is due to the fact most version range checks require these
// semantics.
//
// WARNING: invalid semantic versions are treated similarly to the "0.0.0"
// version.
func (version Version) IsInRange(inclusiveLowerBoundary, exclusiveUpperBoundary Version) (isInRange bool) {
	return !version.IsLessThan(inclusiveLowerBoundary) &&
		version.IsLessThan(exclusiveUpperBoundary)
}

// IsLessThan returns true if the receiver version is less than the specified
// version comparing major, minor, patch and prerelease versions according to
// the semantic version specification.
//
// WARNING: the comparison ignores the build metadata part of the versions.
//
// WARNING: invalid semantic versions are treated similarly to the "0.0.0"
// version.
func (version Version) IsLessThan(otherVersion Version) (isLessThan bool) {
	return version.Compare(otherVersion) == ComparedLess
}

// IsValid returns true if the version is a valid semantic version, false
// otherwise.
func (version Version) IsValid() (isValid bool) {
	return VersionRegex.MatchString(version.String())
}

// Major returns the major version part of a semantic version.
//
// WARNING: for an invalid semantic version this function returns 0.
func (version Version) Major() (major int) {
	major, _, _, _, _ = version.Parts() // nolint:dogsled // Note: intentional performance/usability choice.

	return major
}

// Minor returns the minor version part of a semantic version.
//
// WARNING: for an invalid semantic version this function returns 0.
func (version Version) Minor() (minor int) {
	_, minor, _, _, _ = version.Parts() // nolint:dogsled // Note: intentional performance/usability choice.

	return minor
}

// Parts returns the different parts of a semantic version as separate fields.
//
// WARNING: for an invalid semantic version this function returns the default 0
// and nil values.
//
// WARNING: the returned prereleases part does not contain the leading hyphen.
// The returned builds part does not contain the leading plus.
func (version Version) Parts() (major, minor, patch int, prereleases, builds []string) {
	rawMajor, rawMinor, rawPatch, rawPrerelease, rawBuild := version.RawParts() // Note: valid semantic version.

	major, _ = strconv.Atoi(rawMajor) // Note: the regex ensures no error here.
	minor, _ = strconv.Atoi(rawMinor) // Note: the regex ensures no error here.
	patch, _ = strconv.Atoi(rawPatch) // Note: the regex ensures no error here.

	if rawPrerelease != "" {
		prereleases = strings.Split(rawPrerelease, ".")
	}

	if rawBuild != "" {
		builds = strings.Split(rawBuild, ".")
	}

	return major, minor, patch, prereleases, builds
}

// Patch returns the patch version part of a semantic version.
//
// WARNING: for an invalid semantic version this function returns 0.
func (version Version) Patch() (patch int) {
	_, _, patch, _, _ = version.Parts() // nolint:dogsled // Note: intentional performance/usability choice.

	return patch
}

// Prerelease returns the prerelease part of a semantic version as a string.
//
// WARNING: for an invalid semantic version this function returns the empty
// string.
//
// WARNING: the returned prerelease part does not contain the leading hyphen.
func (version Version) Prerelease() (prerelease string) {
	_, _, _, prerelease, _ = version.RawParts() // nolint:dogsled // Note: intentional performance/usability choice.

	return prerelease
}

// Prereleases returns the prerelease parts of a semantic version as a string
// slice.
//
// WARNING: for an invalid semantic version this function returns nil.
//
// WARNING: the returned prerelease parts do not contain the leading hyphen.
func (version Version) Prereleases() (prereleases []string) {
	_, _, _, prereleases, _ = version.Parts() // nolint:dogsled // Note: intentional performance/usability choice.

	return prereleases
}

// RawParts returns the possible known parts of a semantic version as raw
// strings.
//
// WARNING: for an invalid semantic version this function returns the parts of
// the "0.0.0" valid semantic version.
func (version Version) RawParts() (major, minor, patch, prerelease, build string) {
	submatches := VersionRegex.FindStringSubmatch(version.String())
	if len(submatches) != 6 {
		return "0", "0", "0", "", ""
	}

	return submatches[1], submatches[2], submatches[3], submatches[4], submatches[5]
}

// String returns the string representation of the version.
func (version Version) String() (versionString string) {
	return string(version)
}

// CheckVersionString checks whether the specified string would be a valid
// version and returns error if it would not.
func CheckVersionString(candidate string) (err error) {
	if !VersionRegex.MatchString(candidate) {
		return ErrorInvalidVersion(candidate)
	}

	return nil
}

// IsValidVersionString determines whether the specified string would be a valid
// version.
func IsValidVersionString(candidate string) (isValid bool) {
	if !VersionRegex.MatchString(candidate) {
		return false
	}

	return true
}
