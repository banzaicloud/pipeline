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

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/pkg/any"
	"github.com/banzaicloud/pipeline/pkg/brn"
)

// FeatureManager implements the DNS feature manager
type FeatureManager struct {
	clusterOrgIDGetter ClusterOrgIDGetter
	clusterUIDGetter   ClusterUIDGetter
	config             Config
}

// ClusterOrgIDGetter can be used to get the ID of the organization a cluster belongs to
type ClusterOrgIDGetter interface {
	GetClusterOrgID(ctx context.Context, clusterID uint) (uint, error)
}

// ClusterUIDGetter can be used to get the UID of a cluster
type ClusterUIDGetter interface {
	GetClusterUID(ctx context.Context, clusterID uint) (string, error)
}

// NewFeatureManager returns a DNS feature manager
func NewFeatureManager(clusterOrgIDGetter ClusterOrgIDGetter, clusterUIDGetter ClusterUIDGetter, config Config) FeatureManager {
	return FeatureManager{
		clusterOrgIDGetter: clusterOrgIDGetter,
		clusterUIDGetter:   clusterUIDGetter,
		config:             config,
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
func (m FeatureManager) PrepareSpec(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	defaulters := mapStringXform(map[string]any.Transformation{
		"externalDns": mapStringDefaulter(map[string]any.Transformation{
			"txtOwnerId": txtOwnerIDDefaulterXform(func() (string, error) {
				return m.clusterUIDGetter.GetClusterUID(ctx, clusterID)
			}),
		}),
	})
	xform := mapStringXform(map[string]any.Transformation{
		"externalDns": mapStringXform(map[string]any.Transformation{
			"provider": mapStringXform(map[string]any.Transformation{
				"secretId": secretBRNXform(func() (uint, error) {
					return m.clusterOrgIDGetter.GetClusterOrgID(ctx, clusterID)
				}),
			}),
		}),
	})

	res, err := any.Compose(defaulters, xform).Transform(spec)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to transform spec")
	}
	if r, ok := res.(clusterfeature.FeatureSpec); ok {
		return r, nil
	}
	return nil, errors.Errorf("cannot cast type %T as type %T", res, spec)
}

func mapStringXform(transformations map[string]any.Transformation) any.Transformation {
	return any.TransformationFunc(func(o interface{}) (interface{}, error) {
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

func secretBRNXform(getOrgID func() (uint, error)) any.Transformation {
	return any.TransformationFunc(func(secretObj interface{}) (interface{}, error) {
		if secretStr, ok := secretObj.(string); ok {
			orgID, err := getOrgID()
			if err != nil {
				return secretObj, errors.WrapIf(err, "failed to get org ID")
			}

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

func mapStringDefaulter(trasformations map[string]any.Transformation) any.Transformation {
	return any.TransformationFunc(func(o interface{}) (interface{}, error) {
		if m, ok := o.(map[string]interface{}); ok {
			n := make(map[string]interface{}, len(m))
			for k, v := range m {
				n[k] = v
			}

			var errs error
			for k, t := range trasformations {
				v, err := t.Transform(n[k])
				errs = errors.Append(errs, err)
				n[k] = v
			}

			return n, errs
		}
		return o, nil
	})
}

func txtOwnerIDDefaulterXform(getClusterUID func() (string, error)) any.Transformation {
	return any.TransformationFunc(func(txtOwnerIDObj interface{}) (interface{}, error) {
		if txtOwnerIDStr, ok := txtOwnerIDObj.(string); ok && txtOwnerIDStr != "" {
			return txtOwnerIDStr, nil
		}

		uid, err := getClusterUID()
		return uid, errors.WrapIf(err, "failed to get cluster UID")
	})
}
