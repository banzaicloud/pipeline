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

package kubernetes

import "time"

func retry(function func() error, count int, delaySeconds int) error {
	i := 1
	for {
		err := function()
		if err == nil || i == count {
			return err
		}
		time.Sleep(time.Second * time.Duration(delaySeconds))
		i++
	}
}
