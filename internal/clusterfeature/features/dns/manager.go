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
	"encoding/json"
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
)

const (
	featureName = "dns"

	// hardcoded values for externalDns feature
	externalDnsChartVersion = "2.3.3"

	externalDnsChartName = "stable/external-dns"

	externalDnsNamespace = "pipeline-system"

	externalDnsRelease = "dns"
)

// dnsFeatureManager synchronous feature manager
type dnsFeatureManager struct {
	featureRepository clusterfeature.FeatureRepository
	secretStore       features.SecretStore
	clusterGetter     clusterfeatureadapter.ClusterGetter
	clusterService    clusterfeature.ClusterService
	helmService       features.HelmService
	orgDomainService  OrgDomainService

	logger common.Logger
}

// NewDnsFeatureManager builds a new feature manager component
func NewDnsFeatureManager(
	featureRepository clusterfeature.FeatureRepository,
	secretStore features.SecretStore,
	clusterService clusterfeature.ClusterService,
	clusterGetter clusterfeatureadapter.ClusterGetter,
	helmService features.HelmService,
	orgDomainService OrgDomainService,

	logger common.Logger,
) clusterfeature.FeatureManager {
	return &dnsFeatureManager{
		featureRepository: featureRepository,
		secretStore:       secretStore,
		clusterService:    clusterService,
		clusterGetter:     clusterGetter,
		helmService:       helmService,
		orgDomainService:  orgDomainService,
		logger:            logger,
	}
}

func (m *dnsFeatureManager) Details(ctx context.Context, clusterID uint) (*clusterfeature.Feature, error) {
	ctx, err := m.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {

		return nil, err
	}

	feature, err := m.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {

		return nil, err
	}

	if feature == nil {

		return nil, clusterfeature.FeatureNotFoundError{FeatureName: featureName}
	}

	feature, err = m.decorateWithOutput(ctx, clusterID, feature)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to decorate with output")
	}

	return feature, nil
}

func (m *dnsFeatureManager) Name() string {
	return featureName
}

func (m *dnsFeatureManager) Activate(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	if err := m.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := m.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {

		return err
	}

	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	boundSpec, err := m.bindInput(ctx, spec)
	if err != nil {

		return err
	}

	dnsChartValues := &ExternalDnsChartValues{}

	if boundSpec.AutoDns.Enabled {

		if err := m.orgDomainService.EnsureOrgDomain(ctx, clusterID); err != nil {
			logger.Debug("failed to enable autoDNS")

			return errors.WrapIf(err, "failed to register org hosted zone")
		}

		dnsChartValues, err = m.processAutoDNSFeatureValues(ctx, clusterID, boundSpec.AutoDns)
		if err != nil {
			logger.Debug("failed to process autoDNS values")

			return errors.WrapIf(err, "failed to process autoDNS values")
		}
		d, _, _ := m.orgDomainService.GetDomain(ctx, clusterID)

		dnsChartValues.DomainFilters = []string{d}
	}

	if boundSpec.CustomDns.Enabled {

		dnsChartValues, err = m.processCustomDNSFeatureValues(ctx, clusterID, boundSpec.CustomDns)
		if err != nil {
			logger.Debug("failed to process customDNS values")

			return errors.WrapIf(err, "failed to process customDNS values")
		}
	}

	valuesBytes, err := json.Marshal(dnsChartValues)
	if err != nil {
		logger.Debug("failed to marshal values")

		return errors.WrapIf(err, "failed to decode values")
	}

	if err = m.helmService.InstallDeployment(
		ctx,
		clusterID,
		externalDnsNamespace,
		externalDnsChartName,
		externalDnsRelease,
		valuesBytes,
		externalDnsChartVersion,
		false,
	); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	return nil
}

func (m *dnsFeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	dnsSpec, err := m.bindInput(ctx, spec)
	if err != nil {
		return err
	}

	if !dnsSpec.AutoDns.Enabled && !dnsSpec.CustomDns.Enabled {

		return errors.New("none of the autoDNS and customDNS components are enabled")
	}

	if dnsSpec.AutoDns.Enabled && dnsSpec.CustomDns.Enabled {

		return errors.New("only one of the autoDNS and customDNS components can be enabled")
	}

	if dnsSpec.AutoDns.Enabled {

		err := m.validateAutoDNS(dnsSpec.AutoDns)
		if err != nil {

			return err
		}
	}

	if dnsSpec.CustomDns.Enabled {

		err := m.validateCustomDNS(dnsSpec.CustomDns)
		if err != nil {

			return err
		}
	}

	return nil
}

func (m *dnsFeatureManager) Deactivate(ctx context.Context, clusterID uint) error {
	if err := m.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := m.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {

		return err
	}

	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	if err := m.helmService.DeleteDeployment(ctx, clusterID, externalDnsRelease); err != nil {
		logger.Info("failed to delete feature deployment")

		return errors.WrapIf(err, "failed to uninstall feature")
	}

	return nil
}

func (m *dnsFeatureManager) Update(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	if err := m.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	ctx, err := m.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {

		return err
	}

	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	boundSpec, err := m.bindInput(ctx, spec)
	if err != nil {

		return err
	}

	dnsChartValues := &ExternalDnsChartValues{}

	if boundSpec.AutoDns.Enabled {

		if err := m.orgDomainService.EnsureOrgDomain(ctx, clusterID); err != nil {
			logger.Debug("failed to enable autoDNS")

			return errors.WrapIf(err, "failed to register org hosted zone")
		}

		dnsChartValues, err = m.processAutoDNSFeatureValues(ctx, clusterID, boundSpec.AutoDns)
		if err != nil {
			logger.Debug("failed to process autoDNS values")

			return errors.WrapIf(err, "failed to process autoDNS values")
		}
		d, _, _ := m.orgDomainService.GetDomain(ctx, clusterID)

		dnsChartValues.DomainFilters = []string{d}
	}

	if boundSpec.CustomDns.Enabled {

		dnsChartValues, err = m.processCustomDNSFeatureValues(ctx, clusterID, boundSpec.CustomDns)
		if err != nil {
			logger.Debug("failed to process customDNS values")

			return errors.WrapIf(err, "failed to process customDNS values")
		}
	}

	valuesBytes, err := json.Marshal(dnsChartValues)
	if err != nil {
		logger.Debug("failed to marshal values")

		return errors.WrapIf(err, "failed to decode values")
	}

	if _, err = m.featureRepository.UpdateFeatureSpec(ctx, clusterID, featureName, spec); err != nil {
		logger.Debug("failed to update feature spec")

		return err
	}

	if err = m.helmService.UpdateDeployment(ctx,
		clusterID,
		externalDnsNamespace,
		externalDnsChartName,
		externalDnsRelease,
		valuesBytes,
		externalDnsChartVersion); err != nil {
		logger.Debug("failed to update")

		return errors.WrapIf(err, "failed to update feature")
	}

	// feature status set back to active
	if _, err = m.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, clusterfeature.FeatureStatusActive); err != nil {
		logger.Debug("failed to update feature status")

		return err
	}

	logger.Info("successfully updated feature")

	return nil
}

func (m *dnsFeatureManager) validateAutoDNS(autoDns AutoDns) error {
	if !autoDns.Enabled {
		return errors.New("autoDNS config must be set")
	}

	return nil
}

func (m *dnsFeatureManager) validateCustomDNS(customDns CustomDns) error {
	if !customDns.Enabled {
		return errors.New("customDNS config must be set")
	}

	if len(customDns.DomainFilters) < 1 {
		return errors.New("domain filters must be provided")
	}

	if customDns.Provider.Name == "" {
		return errors.New("DNS provider name must be provided")
	}

	if customDns.Provider.SecretID == "" {
		return errors.New("secret ID with DNS provider credentials must be provided")
	}

	return nil
}

func (m *dnsFeatureManager) decorateWithOutput(ctx context.Context, clusterID uint, feature *clusterfeature.Feature) (*clusterfeature.Feature, error) {
	if feature == nil {

		return nil, errors.NewWithDetails("no spec provided")
	}

	fSpec, err := m.bindInput(ctx, feature.Spec)
	if err != nil {

		return nil, errors.WrapIf(err, "failed to decode feature spec")
	}

	if fSpec.AutoDns.Enabled {
		domain, _, _ := m.orgDomainService.GetDomain(ctx, clusterID)

		c, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get cluster for output generation")
		}

		clusterDomain := fmt.Sprintf("%s.%s", c.GetName(), domain)

		type zoneInfo struct {
			Zone          string `json:"zone"`
			ClusterDomain string `json:"clusterDomain"`
		}

		// decorate the feature with the output
		type output struct {
			AutoDns zoneInfo `json:"autoDns"`
		}

		o := output{
			AutoDns: zoneInfo{
				Zone:          domain,
				ClusterDomain: clusterDomain,
			},
		}

		var out map[string]interface{}
		j, _ := json.Marshal(&o)

		err = json.Unmarshal(j, &out)
		if err != nil {
			return nil, errors.WrapIf(err, "failed generate output")
		}

		feature.Output = out
	}

	return feature, nil
}

func (m *dnsFeatureManager) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cl, err := m.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		org, err := auth.GetOrganizationById(cl.GetOrganizationId())
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get organization by ID")
		}
		ctx = context.WithValue(ctx, auth.CurrentOrganization, org)
	}
	return ctx, nil
}
