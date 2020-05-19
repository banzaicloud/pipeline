// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package helm

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockOrgService is an autogenerated mock for the OrgService type.
type MockOrgService struct {
	mock.Mock
}

// GetOrgNameByOrgID provides a mock function.
func (_m *MockOrgService) GetOrgNameByOrgID(ctx context.Context, orgID uint) (string, error) {
	ret := _m.Called(ctx, orgID)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, uint) string); ok {
		r0 = rf(ctx, orgID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint) error); ok {
		r1 = rf(ctx, orgID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockEnvResolver is an autogenerated mock for the EnvResolver type.
type MockEnvResolver struct {
	mock.Mock
}

// ResolveHelmEnv provides a mock function.
func (_m *MockEnvResolver) ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error) {
	ret := _m.Called(ctx, organizationID)

	var r0 HelmEnv
	if rf, ok := ret.Get(0).(func(context.Context, uint) HelmEnv); ok {
		r0 = rf(ctx, organizationID)
	} else {
		r0 = ret.Get(0).(HelmEnv)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint) error); ok {
		r1 = rf(ctx, organizationID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ResolvePlatformEnv provides a mock function.
func (_m *MockEnvResolver) ResolvePlatformEnv(ctx context.Context) (HelmEnv, error) {
	ret := _m.Called(ctx)

	var r0 HelmEnv
	if rf, ok := ret.Get(0).(func(context.Context) HelmEnv); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(HelmEnv)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockService is an autogenerated mock for the Service type.
type MockService struct {
	mock.Mock
}

// AddRepository provides a mock function.
func (_m *MockService) AddRepository(ctx context.Context, organizationID uint, repository Repository) error {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) error); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckRelease provides a mock function.
func (_m *MockService) CheckRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) (string, error) {
	ret := _m.Called(ctx, organizationID, clusterID, releaseName, options)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, string, Options) string); ok {
		r0 = rf(ctx, organizationID, clusterID, releaseName, options)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, uint, string, Options) error); ok {
		r1 = rf(ctx, organizationID, clusterID, releaseName, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteRelease provides a mock function.
func (_m *MockService) DeleteRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) error {
	ret := _m.Called(ctx, organizationID, clusterID, releaseName, options)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, string, Options) error); ok {
		r0 = rf(ctx, organizationID, clusterID, releaseName, options)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteRepository provides a mock function.
func (_m *MockService) DeleteRepository(ctx context.Context, organizationID uint, repoName string) error {
	ret := _m.Called(ctx, organizationID, repoName)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, string) error); ok {
		r0 = rf(ctx, organizationID, repoName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetChart provides a mock function.
func (_m *MockService) GetChart(ctx context.Context, organizationID uint, chartFilter ChartFilter, options Options) (chartDetails ChartDetails, err error) {
	ret := _m.Called(ctx, organizationID, chartFilter, options)

	var r0 ChartDetails
	if rf, ok := ret.Get(0).(func(context.Context, uint, ChartFilter, Options) ChartDetails); ok {
		r0 = rf(ctx, organizationID, chartFilter, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ChartDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, ChartFilter, Options) error); ok {
		r1 = rf(ctx, organizationID, chartFilter, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRelease provides a mock function.
func (_m *MockService) GetRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) (Release, error) {
	ret := _m.Called(ctx, organizationID, clusterID, releaseName, options)

	var r0 Release
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, string, Options) Release); ok {
		r0 = rf(ctx, organizationID, clusterID, releaseName, options)
	} else {
		r0 = ret.Get(0).(Release)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, uint, string, Options) error); ok {
		r1 = rf(ctx, organizationID, clusterID, releaseName, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetReleaseResources provides a mock function.
func (_m *MockService) GetReleaseResources(ctx context.Context, organizationID uint, clusterID uint, release Release, options Options) ([]ReleaseResource, error) {
	ret := _m.Called(ctx, organizationID, clusterID, release, options)

	var r0 []ReleaseResource
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, Release, Options) []ReleaseResource); ok {
		r0 = rf(ctx, organizationID, clusterID, release, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ReleaseResource)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, uint, Release, Options) error); ok {
		r1 = rf(ctx, organizationID, clusterID, release, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetReleases provides a mock function.
func (_m *MockService) GetReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) (releaseList []DetailedRelease, err error) {
	ret := _m.Called(ctx, organizationID, clusterID, filters, options)

	var r0 []DetailedRelease
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, ReleaseFilter, Options) []DetailedRelease); ok {
		r0 = rf(ctx, organizationID, clusterID, filters, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]DetailedRelease)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, uint, ReleaseFilter, Options) error); ok {
		r1 = rf(ctx, organizationID, clusterID, filters, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InstallRelease provides a mock function.
func (_m *MockService) InstallRelease(ctx context.Context, organizationID uint, clusterID uint, release Release, options Options) error {
	ret := _m.Called(ctx, organizationID, clusterID, release, options)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, Release, Options) error); ok {
		r0 = rf(ctx, organizationID, clusterID, release, options)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ListCharts provides a mock function.
func (_m *MockService) ListCharts(ctx context.Context, organizationID uint, filter ChartFilter, options Options) (charts []interface{}, err error) {
	ret := _m.Called(ctx, organizationID, filter, options)

	var r0 []interface{}
	if rf, ok := ret.Get(0).(func(context.Context, uint, ChartFilter, Options) []interface{}); ok {
		r0 = rf(ctx, organizationID, filter, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, ChartFilter, Options) error); ok {
		r1 = rf(ctx, organizationID, filter, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListReleases provides a mock function.
func (_m *MockService) ListReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) ([]Release, error) {
	ret := _m.Called(ctx, organizationID, clusterID, filters, options)

	var r0 []Release
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, ReleaseFilter, Options) []Release); ok {
		r0 = rf(ctx, organizationID, clusterID, filters, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Release)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, uint, ReleaseFilter, Options) error); ok {
		r1 = rf(ctx, organizationID, clusterID, filters, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListRepositories provides a mock function.
func (_m *MockService) ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error) {
	ret := _m.Called(ctx, organizationID)

	var r0 []Repository
	if rf, ok := ret.Get(0).(func(context.Context, uint) []Repository); ok {
		r0 = rf(ctx, organizationID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Repository)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint) error); ok {
		r1 = rf(ctx, organizationID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ModifyRepository provides a mock function.
func (_m *MockService) ModifyRepository(ctx context.Context, organizationID uint, repository Repository) error {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) error); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateRepository provides a mock function.
func (_m *MockService) UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) error); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpgradeRelease provides a mock function.
func (_m *MockService) UpgradeRelease(ctx context.Context, organizationID uint, clusterID uint, release Release, options Options) error {
	ret := _m.Called(ctx, organizationID, clusterID, release, options)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, Release, Options) error); ok {
		r0 = rf(ctx, organizationID, clusterID, release, options)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockEnvService is an autogenerated mock for the EnvService type.
type MockEnvService struct {
	mock.Mock
}

// AddRepository provides a mock function.
func (_m *MockEnvService) AddRepository(ctx context.Context, helmEnv HelmEnv, repository Repository) error {
	ret := _m.Called(ctx, helmEnv, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, Repository) error); ok {
		r0 = rf(ctx, helmEnv, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckReleaseCharts provides a mock function.
func (_m *MockEnvService) CheckReleaseCharts(ctx context.Context, helmEnv HelmEnv, releases []Release) (map[string]bool, error) {
	ret := _m.Called(ctx, helmEnv, releases)

	var r0 map[string]bool
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, []Release) map[string]bool); ok {
		r0 = rf(ctx, helmEnv, releases)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]bool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, HelmEnv, []Release) error); ok {
		r1 = rf(ctx, helmEnv, releases)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteRepository provides a mock function.
func (_m *MockEnvService) DeleteRepository(ctx context.Context, helmEnv HelmEnv, repoName string) error {
	ret := _m.Called(ctx, helmEnv, repoName)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, string) error); ok {
		r0 = rf(ctx, helmEnv, repoName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EnsureEnv provides a mock function.
func (_m *MockEnvService) EnsureEnv(ctx context.Context, helmEnv HelmEnv, defaultRepos []Repository) (HelmEnv, bool, error) {
	ret := _m.Called(ctx, helmEnv, defaultRepos)

	var r0 HelmEnv
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, []Repository) HelmEnv); ok {
		r0 = rf(ctx, helmEnv, defaultRepos)
	} else {
		r0 = ret.Get(0).(HelmEnv)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(context.Context, HelmEnv, []Repository) bool); ok {
		r1 = rf(ctx, helmEnv, defaultRepos)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, HelmEnv, []Repository) error); ok {
		r2 = rf(ctx, helmEnv, defaultRepos)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetChart provides a mock function.
func (_m *MockEnvService) GetChart(ctx context.Context, helmEnv HelmEnv, chartFilter ChartFilter) (chartDetails ChartDetails, err error) {
	ret := _m.Called(ctx, helmEnv, chartFilter)

	var r0 ChartDetails
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, ChartFilter) ChartDetails); ok {
		r0 = rf(ctx, helmEnv, chartFilter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ChartDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, HelmEnv, ChartFilter) error); ok {
		r1 = rf(ctx, helmEnv, chartFilter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListCharts provides a mock function.
func (_m *MockEnvService) ListCharts(ctx context.Context, helmEnv HelmEnv, chartFilter ChartFilter) (chartList []interface{}, err error) {
	ret := _m.Called(ctx, helmEnv, chartFilter)

	var r0 []interface{}
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, ChartFilter) []interface{}); ok {
		r0 = rf(ctx, helmEnv, chartFilter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, HelmEnv, ChartFilter) error); ok {
		r1 = rf(ctx, helmEnv, chartFilter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListRepositories provides a mock function.
func (_m *MockEnvService) ListRepositories(ctx context.Context, helmEnv HelmEnv) (repos []Repository, err error) {
	ret := _m.Called(ctx, helmEnv)

	var r0 []Repository
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv) []Repository); ok {
		r0 = rf(ctx, helmEnv)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Repository)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, HelmEnv) error); ok {
		r1 = rf(ctx, helmEnv)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PatchRepository provides a mock function.
func (_m *MockEnvService) PatchRepository(ctx context.Context, helmEnv HelmEnv, repository Repository) error {
	ret := _m.Called(ctx, helmEnv, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, Repository) error); ok {
		r0 = rf(ctx, helmEnv, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateRepository provides a mock function.
func (_m *MockEnvService) UpdateRepository(ctx context.Context, helmEnv HelmEnv, repository Repository) error {
	ret := _m.Called(ctx, helmEnv, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, HelmEnv, Repository) error); ok {
		r0 = rf(ctx, helmEnv, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockStore is an autogenerated mock for the Store type.
type MockStore struct {
	mock.Mock
}

// Create provides a mock function.
func (_m *MockStore) Create(ctx context.Context, organizationID uint, repository Repository) error {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) error); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete provides a mock function.
func (_m *MockStore) Delete(ctx context.Context, organizationID uint, repository Repository) error {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) error); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function.
func (_m *MockStore) Get(ctx context.Context, organizationID uint, repository Repository) (Repository, error) {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 Repository
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) Repository); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Get(0).(Repository)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint, Repository) error); ok {
		r1 = rf(ctx, organizationID, repository)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function.
func (_m *MockStore) List(ctx context.Context, organizationID uint) ([]Repository, error) {
	ret := _m.Called(ctx, organizationID)

	var r0 []Repository
	if rf, ok := ret.Get(0).(func(context.Context, uint) []Repository); ok {
		r0 = rf(ctx, organizationID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Repository)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint) error); ok {
		r1 = rf(ctx, organizationID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function.
func (_m *MockStore) Update(ctx context.Context, organizationID uint, repository Repository) error {
	ret := _m.Called(ctx, organizationID, repository)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, Repository) error); ok {
		r0 = rf(ctx, organizationID, repository)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockSecretStore is an autogenerated mock for the SecretStore type.
type MockSecretStore struct {
	mock.Mock
}

// CheckPasswordSecret provides a mock function.
func (_m *MockSecretStore) CheckPasswordSecret(ctx context.Context, secretID string) error {
	ret := _m.Called(ctx, secretID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, secretID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckTLSSecret provides a mock function.
func (_m *MockSecretStore) CheckTLSSecret(ctx context.Context, secretID string) error {
	ret := _m.Called(ctx, secretID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, secretID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ResolvePasswordSecrets provides a mock function.
func (_m *MockSecretStore) ResolvePasswordSecrets(ctx context.Context, secretID string) (PasswordSecret, error) {
	ret := _m.Called(ctx, secretID)

	var r0 PasswordSecret
	if rf, ok := ret.Get(0).(func(context.Context, string) PasswordSecret); ok {
		r0 = rf(ctx, secretID)
	} else {
		r0 = ret.Get(0).(PasswordSecret)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, secretID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ResolveTlsSecrets provides a mock function.
func (_m *MockSecretStore) ResolveTlsSecrets(ctx context.Context, secretID string) (TlsSecret, error) {
	ret := _m.Called(ctx, secretID)

	var r0 TlsSecret
	if rf, ok := ret.Get(0).(func(context.Context, string) TlsSecret); ok {
		r0 = rf(ctx, secretID)
	} else {
		r0 = ret.Get(0).(TlsSecret)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, secretID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
