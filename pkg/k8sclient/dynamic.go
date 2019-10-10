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

package k8sclient

import (
	"bytes"
	"context"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynamicFileClient interacts with a cluster with file manifests.
type DynamicFileClient struct {
	client client.Client
}

// NewDynamicFileClient returns a new DynamicFileClient.
func NewDynamicFileClient(client client.Client) DynamicFileClient {
	return DynamicFileClient{
		client: client,
	}
}

// Create iterates a set of YAML documents and calls client.Create on them.
func (c DynamicFileClient) Create(ctx context.Context, file []byte) error {
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(file), 4096)

	var objects []unstructured.Unstructured

	for {
		var object unstructured.Unstructured
		if err := decoder.Decode(&object); err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		objects = append(objects, object)
	}

	for _, object := range objects {
		err := c.client.Create(ctx, &object)
		if err != nil {
			return err
		}
	}

	return nil
}
