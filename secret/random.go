// Copyright © 2018 Banzai Cloud
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

package secret

import (
	"fmt"

	"github.com/aokoli/goutils"
)

// DefaultPasswordFormat is the format of passwords if not specified otherwise
const DefaultPasswordFormat = "randAlphaNum,12"

//RandomString creates a random string whose length is the number of characters specified.
func RandomString(genType string, length int) (res string, err error) {
	switch genType {
	case "randAlphaNum":
		res, err = goutils.RandomAlphaNumeric(length)
	case "randAlpha":
		res, err = goutils.RandomAlphabetic(length)
	case "randNumeric":
		res, err = goutils.RandomNumeric(length)
	case "randAscii":
		res, err = goutils.RandomAscii(length)
	default:
		return res, fmt.Errorf("unsupported random type: %s", genType)
	}
	return

}
