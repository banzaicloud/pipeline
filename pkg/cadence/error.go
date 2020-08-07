// Copyright © 2019 Banzai Cloud
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

package cadence

import (
	"errors"
	"fmt"
)

// UnwrapError returns a new detailed error based on the wrapped string, or the
// original error otherwise
func UnwrapError(err error) error {
	if err == nil {
		return nil
	}

	var detailedError DetailedError
	if errors.As(err, &detailedError) {
		var details string
		if detailedError.Details(&details) == nil {
			return fmt.Errorf("%s: %s", err.Error(), details) // Note: detailed errors return their reason on Error().
		}
	}

	return err
}
