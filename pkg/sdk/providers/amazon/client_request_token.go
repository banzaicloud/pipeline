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

package amazon

import (
	"regexp"
	"strings"
)

// clientRequestTokenNotAllowedCharacters is the regular expression matching
// characters not allowed in a client request token.
//
// Regular expression: https://regex101.com/r/yG9DFU/1
//
// Source: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html
var clientRequestTokenNotAllowedCharactersRegexp = regexp.MustCompile("[^0-9a-zA-Z-]") // nolint:gochecknoglobals // Note: best way I know to avoid multiple initializations of single regular expression.

// NewClientRequestToken creates an AWS CloudFormation client request token from
// the specified elements.
//
// 1. The elements are joint by a dash separator.
// 2. Not allowed characters are replaced by dashes.
// 3. The prefix dashes are trimmed.
// 4. The resulting string is truncated to 64 characters.
//
// Source: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html
func NewNormalizedClientRequestToken(elements ...string) (normalizedClientRequestToken string) {
	normalizedClientRequestToken = strings.Join(elements, "-")
	normalizedClientRequestToken = clientRequestTokenNotAllowedCharactersRegexp.ReplaceAllString(
		normalizedClientRequestToken,
		"-",
	)
	normalizedClientRequestToken = strings.TrimLeft(normalizedClientRequestToken, "-")

	if len(normalizedClientRequestToken) > 64 {
		normalizedClientRequestToken = normalizedClientRequestToken[:64]
	}

	return normalizedClientRequestToken
}
