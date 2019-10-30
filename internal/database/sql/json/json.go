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

package json

import (
	"database/sql/driver"
	"encoding/json"
)

// Scan decodes JSON data from the string or byte slice in d into the value pointed to by v and returns any errors.
func Scan(d interface{}, v interface{}) error {
	// if d is a string the cast to byte slice would fail, but we can work around that
	if s, ok := d.(string); ok {
		d = []byte(s)
	}
	return json.Unmarshal(d.([]byte), v)
}

// Value takes a value and returns it JSON encoded as a suitable database/sql/driver.Value or returns any errors.
func Value(v interface{}) (driver.Value, error) {
	return json.Marshal(v)
}
