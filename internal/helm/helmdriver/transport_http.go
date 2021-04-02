// Copyright © 2020 Banzai Cloud
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

package helmdriver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/helm"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodPost).Path("/repos").Handler(kithttp.NewServer(
		endpoints.AddRepository,
		decodeAddRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/repos").Handler(kithttp.NewServer(
		endpoints.ListRepositories,
		decodeListRepositoriesHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeListRepositoriesHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodDelete).Path("/repos/{name}").Handler(kithttp.NewServer(
		endpoints.DeleteRepository,
		decodeDeleteRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeDeleteRepositoryHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodPut).Path("/repos/{name}").Handler(kithttp.NewServer(
		endpoints.ModifyRepository,
		decodeModifyRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodPut).Path("/repos/{name}/update").Handler(kithttp.NewServer(
		endpoints.UpdateRepository,
		decodeUpdateRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/charts").Handler(kithttp.NewServer(
		endpoints.ListCharts,
		decodeListChartsHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeListChartsHTTPResponse, errorEncoder),
		options...,
	))

	// TODO fix the path after migrating to h3 (use chartS instead of chart) - backwards  compatibility!
	router.Methods(http.MethodGet).Path("/chart/{reponame}/{name}").Handler(kithttp.NewServer(
		endpoints.GetChart,
		decodeChartDetailsHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeChartDetailsHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/cluster-charts").Handler(kithttp.NewServer(
		endpoints.ListClusterCharts,
		decodeListClusterChartsHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeListClusterChartsHTTPResponse, errorEncoder),
		options...,
	))
}

func RegisterReleaserHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodPost).Path("").Handler(kithttp.NewServer(
		endpoints.InstallRelease,
		decodeInstallReleaseHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeInstallReleaseHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodDelete).Path("/{name}").Handler(kithttp.NewServer(
		endpoints.DeleteRelease,
		decodeDeleteReleaseHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodPut).Path("/{name}").Handler(kithttp.NewServer(
		endpoints.UpgradeRelease,
		decodeUpgradeReleaseHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeUpgradeReleaseHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/{name}").Handler(kithttp.NewServer(
		endpoints.GetRelease,
		decodeGetReleaseHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeGetReleaseHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/{name}/resources").Handler(kithttp.NewServer(
		endpoints.GetReleaseResources,
		decodeGetReleaseResourcesHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeGetReleaseResourcesHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodHead).Path("/{name}").Handler(kithttp.NewServer(
		endpoints.CheckRelease,
		decodeCheckReleaseHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeCheckReleaseHTTPResponse, errorEncoder),
		options...,
	))
}

func RegisterRestAPI(endpoints RestAPIEndpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.GetReleases,
		decodeGetReleasesHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeGetReleasesHTTPResponse, errorEncoder),
		options...,
	))
}

func decodeInstallReleaseHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractUintParamFromRequest("orgId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode add repository request")
	}

	clusterID, e := extractUintParamFromRequest("clusterId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode add repository request")
	}

	var request pipeline.CreateUpdateDeploymentRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode request")
	}

	return InstallReleaseRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		ReleaseInput: helm.Release{
			ReleaseName: request.ReleaseName,
			ChartName:   request.Name,
			Namespace:   request.Namespace,
			Values:      request.Values,
			Version:     request.Version,
		},
		Options: helm.Options{
			DryRun:       request.DryRun,
			GenerateName: request.ReleaseName == "",
			Wait:         request.Wait,
			Namespace:    request.Namespace,
		},
	}, nil
}

func encodeInstallReleaseHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(InstallReleaseResponse)
	if !ok {
		return errors.NewWithDetails("failed to encode release resources response")
	}

	if resp.Err != nil {
		return errors.NewWithDetails("failed to retrieve release resources")
	}

	// backwards compatibility!
	oldResponse := struct {
		ReleaseName string                 `json:"releaseName"`
		Notes       string                 `json:"notes"`
		Resources   []helm.ReleaseResource `json:"resources"`
	}{
		ReleaseName: resp.Release.ReleaseName,
		Notes:       resp.Release.ReleaseInfo.Notes,
		Resources:   resp.Release.ReleaseResources,
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, kitxhttp.WithStatusCode(oldResponse, http.StatusCreated))
}

func decodeUpgradeReleaseHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractUintParamFromRequest("orgId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode add repository request")
	}

	clusterID, e := extractUintParamFromRequest("clusterId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode add repository request")
	}

	var request pipeline.CreateUpdateDeploymentRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode request")
	}

	return UpgradeReleaseRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		ReleaseInput: helm.Release{
			ReleaseName: request.ReleaseName,
			ChartName:   request.Name,
			Namespace:   request.Namespace,
			Values:      request.Values,
			Version:     request.Version,
		}, Options: helm.Options{
			DryRun:       request.DryRun,
			GenerateName: request.ReleaseName == "",
			Wait:         request.Wait,
		},
	}, nil
}

func encodeUpgradeReleaseHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(UpgradeReleaseResponse)
	if !ok {
		return errors.NewWithDetails("failed to encode release resources response")
	}

	if resp.Err != nil {
		return errors.NewWithDetails("failed to retrieve release resources")
	}

	// backwards compatibility!
	oldResponse := struct {
		ReleaseName string                 `json:"releaseName"`
		Notes       string                 `json:"notes"`
		Resources   []helm.ReleaseResource `json:"resources"`
	}{
		ReleaseName: resp.Release.ReleaseName,
		Notes:       resp.Release.ReleaseInfo.Notes,
		// Resources:   resp.Release.ReleaseResources,
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, kitxhttp.WithStatusCode(oldResponse, http.StatusCreated))
}

func decodeAddRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractUintParamFromRequest("orgId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode add repository request")
	}

	var request pipeline.HelmReposAddRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode request")
	}

	return AddRepositoryRequest{
		OrganizationID: orgID,
		Repository: helm.Repository{
			Name:             request.Name,
			URL:              request.Url,
			PasswordSecretID: request.PasswordSecretRef,
			TlsSecretID:      request.TlsSecretRef,
		},
	}, nil
}

func decodeModifyRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractUintParamFromRequest("orgId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode modify repository request")
	}

	repoName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode modify repository request")
	}

	var request pipeline.HelmReposAddRequest

	dErr := json.NewDecoder(r.Body).Decode(&request)
	if dErr != nil {
		return nil, errors.WrapIf(dErr, "failed to decode modify repository request")
	}

	return ModifyRepositoryRequest{
		OrganizationID: orgID,
		Repository: helm.Repository{
			Name:             repoName,
			URL:              request.Url,
			PasswordSecretID: request.PasswordSecretRef,
			TlsSecretID:      request.TlsSecretRef,
		},
	}, nil
}

func decodeUpdateRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractUintParamFromRequest("orgId", r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode update repository request")
	}

	repoName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode update repository request")
	}

	return UpdateRepositoryRequest{
		OrganizationID: orgID,
		Repository: helm.Repository{
			Name: repoName,
		},
	}, nil
}

func decodeListRepositoriesHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to decode list request")
	}

	return ListRepositoriesRequest{OrganizationID: orgID}, nil
}

func encodeListRepositoriesHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListRepositoriesResponse)
	list := make([]pipeline.HelmRepoListItem, 0, len(resp.Repos))
	for _, repo := range resp.Repos {
		list = append(list, pipeline.HelmRepoListItem{
			Name:              repo.Name,
			Url:               repo.URL,
			PasswordSecretRef: repo.PasswordSecretID,
			TlsSecretRef:      repo.TlsSecretID,
		})
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, list)
}

func decodeDeleteRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list request")
	}

	repoName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list request")
	}

	return DeleteRepositoryRequest{OrganizationID: orgID, RepoName: repoName}, nil
}

func encodeDeleteRepositoryHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(DeleteRepositoryResponse)
	if ok && resp.Err == nil {
		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, resp)
}

func decodeDeleteReleaseHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode delete release request")
	}

	clusterID, err := extractUintParamFromRequest("clusterId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode delete release request")
	}

	releaseName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode delete release request")
	}

	// namespace is passed as query parameter - quickfix, revisit this and decide on a final solution (eg introduce the namespace rest resource)
	namespace := r.URL.Query().Get("namespace")

	return DeleteReleaseRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		ReleaseName:    releaseName,
		Options: helm.Options{
			Namespace: namespace,
		},
	}, nil
}

func encodeGetReleaseResourcesHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(GetReleaseResourcesResponse)
	if !ok {
		return errors.NewWithDetails("failed to encode release resources response")
	}

	if resp.Err != nil {
		return errors.NewWithDetails("failed to retrieve release resources")
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.R0)
}

func decodeCheckReleaseHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	clusterID, err := extractUintParamFromRequest("clusterId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	releaseName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	return CheckReleaseRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		ReleaseName:    releaseName,
	}, nil
}

func encodeCheckReleaseHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	release, ok := response.(CheckReleaseResponse)
	if !ok {
		return errors.New("invalid  release list response")
	}

	if release.Err != nil {
		return errors.WrapIf(release.Err, "failed to retrieve releases")
	}

	// TODO add this to the api spec
	resp := pkgHelm.DeploymentStatusResponse{
		Status:  http.StatusOK,
		Message: release.R0,
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, resp)
}

func decodeGetReleaseHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	clusterID, err := extractUintParamFromRequest("clusterId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	releaseName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	// namespace is passed as query parameter - quickfix, revisit this and decide on a final solution (eg introduce the namespace rest resource)
	namespace := r.URL.Query().Get("namespace")

	return GetReleaseRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		ReleaseName:    releaseName,
		Options: helm.Options{
			Namespace: namespace,
		},
	}, nil
}

func decodeGetReleaseResourcesHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	clusterID, err := extractUintParamFromRequest("clusterId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	releaseName, err := extractStringParamFromRequest("name", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get release request")
	}

	return GetReleaseResourcesRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		Release: helm.Release{
			ReleaseName: releaseName,
		},
	}, nil
}

func encodeGetReleaseHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	release, ok := response.(GetReleaseResponse)
	if !ok {
		return errors.New("invalid  release list response")
	}

	if release.Err != nil {
		return errors.WrapIf(release.Err, "failed to retrieve releases")
	}

	mergedValues := chartutil.CoalesceTables(release.R0.Values, release.R0.ReleaseInfo.Values)

	resp := pipeline.GetDeploymentResponse{
		ReleaseName:  release.R0.ReleaseName,
		Chart:        release.R0.ChartName, // TODO what's this
		ChartName:    release.R0.ChartName,
		ChartVersion: release.R0.Version,
		Namespace:    release.R0.Namespace,
		Version:      release.R0.ReleaseVersion,
		UpdatedAt:    release.R0.ReleaseInfo.LastDeployed.String(),
		Status:       release.R0.ReleaseInfo.Status,
		CreatedAt:    release.R0.ReleaseInfo.FirstDeployed.String(),
		Notes:        release.R0.ReleaseInfo.Notes,
		Values:       mergedValues,
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, resp)
}

func decodeGetReleasesHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list release request")
	}

	clusterID, err := extractUintParamFromRequest("clusterId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list release request")
	}

	queryData := struct {
		TagFilters []string `json:"tag" mapstructure:"tag"`
		Filters    []string `json:"filter,omitempty" mapstructure:"filter"`
	}{}
	if err := mapstructure.Decode(r.URL.Query(), &queryData); err != nil {
		return nil, errors.WrapIf(err, "failed to decode list release request")
	}

	filter := helm.ReleaseFilter{}
	if len(queryData.TagFilters) > 1 {
		filter.TagFilter = queryData.TagFilters[0]
	}
	if queryData.Filters != nil {
		filter.Filter = &queryData.Filters[0]
	}

	return GetReleasesRestAPIRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		Filters:        filter,
	}, nil
}

func encodeGetReleasesHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	releases, ok := response.(GetReleasesRestAPIResponse)
	if !ok {
		return errors.New("invalid release list response")
	}

	if releases.Err != nil {
		return errors.WrapIf(releases.Err, "failed to retrieve releases")
	}

	resp := make([]pkgHelm.ListDeploymentResponse, 0, len(releases.ReleaseList))
	for _, release := range releases.ReleaseList {
		resp = append(resp, pkgHelm.ListDeploymentResponse{
			Name:         release.ReleaseName,
			Chart:        release.ChartName,
			ChartName:    release.ChartName,
			ChartVersion: release.Version,
			Version:      release.ReleaseVersion,
			UpdatedAt:    release.ReleaseInfo.LastDeployed,
			Status:       release.ReleaseInfo.Status,
			Namespace:    release.Namespace,
			CreatedAt:    release.ReleaseInfo.FirstDeployed,
			Supported:    release.Supported,
			WhiteListed:  release.Whitelisted,
			Rejected:     release.Rejected,
		})
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, resp)
}

func decodeListChartsHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get charts request")
	}

	// WARN: this' struct behavior MUST be is analogue to query api.ChartQuery in order not to break the api
	parsedQuery := helm.ChartFilter{}

	if err := mapstructure.Decode(r.URL.Query(), &parsedQuery); err != nil {
		return nil, errors.WrapIf(err, "failed to decode get charts request")
	}

	return ListChartsRequest{
		OrganizationID: orgID,
		Filter:         parsedQuery,
		Options:        helm.Options{},
	}, nil
}

func encodeListChartsHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	charts, ok := response.(ListChartsResponse)
	if !ok {
		return errors.New("invalid  release list response")
	}

	if charts.Err != nil {
		return errors.WrapIf(charts.Err, "failed to retrieve charts")
	}

	if len(charts.Charts) == 0 {
		return kitxhttp.JSONResponseEncoder(ctx, w, "")
	}

	chartsResponse := make([]interface{}, 0, len(charts.Charts))
	for _, repoCharts := range charts.Charts {
		chartsResponse = append(chartsResponse, repoCharts)
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, chartsResponse)
}

func decodeChartDetailsHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode get charts request")
	}

	var (
		// inline type for binding request path parameters
		pathData struct {
			RepoName string
			Name     string
		}

		// inline type for binding request query parameters
		queryData helm.ChartFilter
	)

	if err := mapstructure.Decode(mux.Vars(r), &pathData); err != nil {
		return nil, errors.WrapIf(err, "failed to decode get chart details path parameters")
	}

	if err := mapstructure.Decode(r.URL.Query(), &queryData); err != nil {
		return nil, errors.WrapIf(err, "failed to decode get chart details query parameters")
	}

	return GetChartRequest{
		OrganizationID: orgID,
		ChartFilter: helm.ChartFilter{
			Repo:    []string{pathData.RepoName},
			Name:    []string{pathData.Name},
			Version: queryData.Version,
			Keyword: queryData.Keyword, // TODO is it used?
		},
	}, nil
}

func encodeChartDetailsHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	chart, ok := response.(GetChartResponse)
	if !ok {
		return errors.New("invalid  release list response")
	}

	if chart.Err != nil {
		return errors.WrapIf(chart.Err, "failed to retrieve charts")
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, chart.ChartDetails)
}

func decodeListClusterChartsHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractUintParamFromRequest("orgId", r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list cluster charts request")
	}

	return ListClusterChartsRequest{
		OrganizationID: orgID,
		Options:        helm.Options{},
	}, nil
}

func encodeListClusterChartsHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	charts, ok := response.(ListClusterChartsResponse)
	if !ok {
		return errors.New("invalid cluster charts list response")
	}

	if charts.Err != nil {
		return errors.WrapIf(charts.Err, "failed to retrieve charts")
	}

	if len(charts.Charts) == 0 {
		return kitxhttp.JSONResponseEncoder(ctx, w, "")
	}

	chartsResponse := make([]interface{}, 0, len(charts.Charts))
	for _, repoCharts := range charts.Charts {
		chartsResponse = append(chartsResponse, repoCharts)
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, chartsResponse)
}

func extractStringParamFromRequest(key string, r *http.Request) (string, error) {
	vars := mux.Vars(r)

	repoName, ok := vars[key]
	if !ok || repoName == "" {
		return "", errors.NewWithDetails("missing path parameter", "param", "name")
	}

	return repoName, nil
}

func extractUintParamFromRequest(key string, r *http.Request) (uint, error) {
	vars := mux.Vars(r)

	strVal, ok := vars[key]
	if !ok || strVal == "" {
		return 0, errors.NewWithDetails("missing path parameter", "param", key)
	}

	uintVal, e := strconv.ParseUint(strVal, 10, 32)
	if e != nil {
		return 0, errors.WrapIff(e, "failed to parse path param: %s, value:  %s", "id", strVal)
	}

	return uint(uintVal), nil
}
