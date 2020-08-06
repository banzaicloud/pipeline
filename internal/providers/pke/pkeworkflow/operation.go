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

package pkeworkflow

import "emperror.dev/errors"

// Note: when generics is implemented (possibly in Golang 2.0) this can be
// replaced with a generic batch operation logic.

// nodePoolOperation defines a function type for running custom operations on a
// node pool and returning the result or alternatively an error.
type nodePoolOperation func(nodePool *NodePool) (err error)

// forEachNodePool implements custom operation batch processing for multiple
// node pools by running the specified operation on every node pool.
//
// Somewhat unidiomatically to Go, The node pool operation allows mutating the
// original node pool in case it should be mutated in place for performance
// reasons. If you want to work on a clean copy, you need to create the copy
// beforehand and specify that as nodePools.
func forEachNodePool(nodePools []NodePool, operation nodePoolOperation) (err error) {
	if len(nodePools) == 0 || operation == nil {
		return nil
	}

	nodePoolErrors := make([]error, 0, len(nodePools))
	for nodePoolIndex := range nodePools {
		nodePoolErrors = append(nodePoolErrors, operation(&nodePools[nodePoolIndex]))
	}
	if err := errors.Combine(nodePoolErrors...); err != nil {
		return err
	}

	return nil
}
