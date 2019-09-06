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

package opaque

// Transformation can transform an opaque value to another
type Transformation interface {
	// Transform performs the transformation
	Transform(interface{}) (interface{}, error)
}

// TransformationFunc wraps a function that implements the transformation
type TransformationFunc func(interface{}) (interface{}, error)

// Transform implements the transformation by delegating to the wrapped function
func (f TransformationFunc) Transform(src interface{}) (interface{}, error) {
	return f(src)
}

// Identity is the identity transformation
const Identity identityTransformation = false

// The underlying type must be one of the scalar types to allow for defining a const instance
type identityTransformation bool

func (identityTransformation) Transform(src interface{}) (interface{}, error) {
	return src, nil
}

// Compose returns the composition of the specified transformations performed in order
func Compose(transformations ...Transformation) Transformation {
	return TransformationFunc(func(o interface{}) (interface{}, error) {
		var err error
		for _, t := range transformations {
			o, err = t.Transform(o)
			if err != nil {
				return o, err
			}
		}
		return o, nil
	})
}
