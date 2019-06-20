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

package errors

// CauseIterator can be used to iterate over some or all of an error's causes
type CauseIterator struct {
	Error error
}

// ForEachWhile calls the provided function with each cause of the error in the iterator while the function returns true
func (i CauseIterator) ForEachWhile(fn func(err error) bool) {
	err := i.Error
	for err != nil {
		fn(err)
		if causer, ok := err.(interface{ Cause() error }); ok {
			err = causer.Cause()
		} else {
			err = nil
		}
	}
}

// Any returns whether the provided predicate is true for any of the causes of the error in the iterator
func (i CauseIterator) Any(predicate func(err error) bool) bool {
	success := false
	i.ForEachWhile(func(err error) bool {
		success = predicate(err)
		return !success
	})
	return success
}
