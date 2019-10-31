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

package dns

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/pkg/brn"
	"github.com/banzaicloud/pipeline/pkg/opaque"
)

// FeatureManager implements the DNS feature manager
type FeatureManager struct {
	config Config
}

// MakeFeatureManager returns a DNS feature manager
func MakeFeatureManager(config Config) FeatureManager {
	return FeatureManager{
		config: config,
	}
}

// Name returns the feature's name
func (FeatureManager) Name() string {
	return FeatureName
}

// GetOutput returns the DNS feature's output
func (m FeatureManager) GetOutput(ctx context.Context, clusterID uint, _ clusterfeature.FeatureSpec) (clusterfeature.FeatureOutput, error) {
	return map[string]interface{}{
		"externalDns": map[string]interface{}{
			"version": m.config.Charts.ExternalDNS.Version,
		},
	}, nil
}

// ValidateSpec validates a DNS feature specification
func (FeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	dnsSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     err.Error(),
		}
	}

	if err := dnsSpec.Validate(); err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     err.Error(),
		}
	}

	return nil
}

// PrepareSpec makes certain preparations to the spec before it's sent to be applied
func (FeatureManager) PrepareSpec(ctx context.Context, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	orgID, ok := auth.GetCurrentOrganizationID(ctx)
	if !ok {
		return nil, errors.New("organization ID missing from context")
	}

	xform := mapStringXform(map[string]opaque.Transformation{
		"externalDns": mapStringXform(map[string]opaque.Transformation{
			"provider": mapStringXform(map[string]opaque.Transformation{
				"secretId": secretBRNXform(orgID),
			}),
		}),
	})

	res, err := xform.Transform(spec)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to transform spec")
	}
	if r, ok := res.(clusterfeature.FeatureSpec); ok {
		return r, nil
	}
	return nil, errors.Errorf("cannot cast type %T as type %T", res, spec)
}

func mapStringXform(transformations map[string]opaque.Transformation) opaque.Transformation {
	return opaque.TransformationFunc(func(o interface{}) (interface{}, error) {
		if m, ok := o.(map[string]interface{}); ok {
			n := make(map[string]interface{}, len(m))
			var errs error
			for k, v := range m {
				if t, ok := transformations[k]; ok {
					w, err := t.Transform(v)
					errs = errors.Append(errs, err)
					n[k] = w
				} else {
					n[k] = v
				}
			}
			return n, errs
		}
		return o, nil
	})
}

func secretBRNXform(orgID uint) opaque.Transformation {
	return opaque.TransformationFunc(func(secretObj interface{}) (interface{}, error) {
		if secretStr, ok := secretObj.(string); ok {
			secretBRN := brn.ResourceName{
				Scheme:         brn.Scheme,
				OrganizationID: orgID,
				ResourceType:   brn.SecretResourceType,
				ResourceID:     secretStr,
			}
			return secretBRN.String(), nil
		}
		return secretObj, nil
	})
}
