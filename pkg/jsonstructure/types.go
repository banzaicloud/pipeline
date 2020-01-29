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

package jsonstructure

// Array represents a JSON array.
type Array = []Value

// Boolean represents a JSON boolean.
type Boolean = bool

// Number represents a JSON number.
type Number = float64

// Object represents a JSON object.
type Object = map[string]Value

// String represents a JSON string.
type String = string

// Value represents any JSON value
type Value = interface{}
