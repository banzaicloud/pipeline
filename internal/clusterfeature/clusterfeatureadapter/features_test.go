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

package clusterfeatureadapter

import (
	"context"
	"testing"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

func TestFeatureSelector_SelectFeature(t *testing.T) {
	tests := []struct {
		name    string
		feature clusterfeature.Feature
		checker func(t *testing.T, fp *clusterfeature.Feature, err error)
	}{
		{
			name: "unsupported feature",
			feature: clusterfeature.Feature{
				Name: "unsupported",
				Spec: map[string]interface{}{},
			},
			checker: func(t *testing.T, fp *clusterfeature.Feature, err error) {
				assert.NotNil(t, err)
				assert.Nil(t, fp)
			},
		},
		{
			name: "supported feature",
			feature: clusterfeature.Feature{
				Name: clusterfeature.ExternalDns,
				Spec: map[string]interface{}{},
			},
			checker: func(t *testing.T, fp *clusterfeature.Feature, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, fp)
				assert.Equal(t, "1.6.2", fp.Spec[clusterfeature.DNSExternalDnsChartVersion])
				assert.Equal(t, "v0.5.11", fp.Spec[clusterfeature.DNSExternalDnsImageVersion])
			},
		},
	}

	fs := clusterfeature.NewFeatureSelector(logur.NewTestLogger())

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fp, e := fs.SelectFeature(context.Background(), test.feature)
			test.checker(t, fp, e)
		})
	}

}
