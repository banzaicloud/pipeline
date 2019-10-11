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

package clustersetup

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

const InitManifestActivityName = "init-manifest"

type InitManifestActivity struct {
	manifest      *template.Template
	clientFactory cluster.DynamicFileClientFactory
}

// NewInitManifestActivity returns a new InitManifestActivity.
func NewInitManifestActivity(manifest *template.Template, clientFactory cluster.DynamicFileClientFactory) InitManifestActivity {
	return InitManifestActivity{
		manifest:      manifest,
		clientFactory: clientFactory,
	}
}

type InitManifestActivityInput struct {
	Cluster      Cluster
	Organization Organization
}

func (a InitManifestActivity) Execute(ctx context.Context, input InitManifestActivityInput) error {
	client, err := a.clientFactory.CreateClient(fmt.Sprintf("id:%d", input.Cluster.ID))
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	err = a.manifest.Execute(&buf, input)
	if err != nil {
		return err
	}

	err = client.Create(ctx, buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
