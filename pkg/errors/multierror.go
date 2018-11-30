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

package errors

import (
	"fmt"
	"strings"
)

type multiError interface {
	Errors() []error
}

type multiErrorWithFormatter struct {
	multiError
}

func NewMultiErrorWithFormatter(err error) error {
	if err, ok := err.(multiError); ok {
		return multiErrorWithFormatter{multiError: err}
	}

	return err
}

func (e multiErrorWithFormatter) Error() string {
	if len(e.Errors()) == 1 {
		return e.Errors()[0].Error()
	}

	points := make([]string, len(e.Errors()))
	for i, er := range e.Errors() {
		points[i] = fmt.Sprintf("* %s", er)
	}

	return fmt.Sprintf("%d errors occurred:\n%s", len(e.Errors()), strings.Join(points, "\n"))
}
