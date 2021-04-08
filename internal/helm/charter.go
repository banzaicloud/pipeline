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

package helm

import (
	"context"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
)

// decouples helm lib types from the api
type ChartDetails map[string]interface{}

// decouples helm lib types from the api
type ChartList = []interface{}

func (c ChartDetails) GetDescription(version string) (string, error) {
	type ChartDetail struct {
		Versions []struct {
			Chart struct {
				Metadata struct {
					Description string
					Version     string
				}
			}
		}
	}

	detail := &ChartDetail{}
	err := mapstructure.Decode(&c, detail)
	if err != nil {
		return "", errors.WrapIf(err, "Unable to decode chart metadata information from chart details")
	}
	for _, cv := range detail.Versions {
		if cv.Chart.Metadata.Version == version {
			return cv.Chart.Metadata.Description, nil
		}
	}
	return "", errors.Errorf("chart version %s not found", version)
}

// charter collects helm chart related operations
// intended  to be embedded into the helm "facade"
type charter interface {
	// List lists charts containing the given term, eventually applying the passed filter
	ListCharts(ctx context.Context, organizationID uint, filter ChartFilter, options Options) (charts ChartList, err error)
	// GetChart retrieves the details for the given chart
	GetChart(ctx context.Context, organizationID uint, chartFilter ChartFilter, options Options) (chartDetails ChartDetails, err error)

	// ListClusterCharts lists the Helm charts (with details) currently
	// available for Pipeline managed clusters.
	ListClusterCharts(ctx context.Context, organizationID uint, options Options) (charts ChartList, err error)

	CheckReleases(ctx context.Context, organizationID uint, releases []Release) (map[string]bool, error)
}

// ChartFilter filter data for chart retrieval
// all fields are slices in order to support forthcoming filtering on multiple values
// Filter values are expected to be used through functions
type ChartFilter struct {
	Repo    []string
	Name    []string
	Version []string
	Keyword []string
}

func (cf ChartFilter) String() string {
	return fmt.Sprintf("repo: %s, chart: %s, version %s", cf.RepoFilter(), cf.StrictNameFilter(), cf.StrictVersionFilter())
}

// RepoFilter gets the string filter eventually trims leading and trailing regexp chars
func (cf ChartFilter) RepoFilter() string {
	// trim regexp markers -if exist
	return strings.TrimSuffix(strings.TrimPrefix(firstOrEmpty(cf.Repo), "^"), "$")
}

// StrictRepoFilter wraps the filter with regexp markers for exact match
func (cf ChartFilter) StrictRepoFilter() string {
	return exactMatchRegexp(cf.RepoFilter())
}

func (cf ChartFilter) StrictNameFilter() string {
	return exactMatchRegexp(firstOrEmpty(cf.Name))
}

func (cf ChartFilter) NameFilter() string {
	return firstOrEmpty(cf.Name)
}

func (cf ChartFilter) StrictVersionFilter() string {
	versionFilter := firstOrEmpty(cf.Version)
	// special cases (backwards comp.)
	if versionFilter == "all" || versionFilter == "latest" {
		return versionFilter
	}
	if versionFilter != "" {
		versionFilter = fmt.Sprintf("%s%s", "^", versionFilter)
	}

	return versionFilter
}

func (cf ChartFilter) VersionFilter() string {
	return firstOrEmpty(cf.Version)
}

func (cf ChartFilter) KeywordFilter() string {
	return firstOrEmpty(cf.Keyword)
}

func firstOrEmpty(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	return slice[0]
}

func exactMatchRegexp(value string) string {
	if value == "" {
		return value
	}

	// the value gets cleaned (aggressively) before applying the exact match regexp boundaries
	value = strings.TrimSuffix(value, "$")
	value = strings.TrimPrefix(value, "^")

	return fmt.Sprintf("%s%s%s", "^", value, "$")
}

// ChartNotFoundError signals that the chart is not found
type ChartNotFoundError struct {
	ChartInfo string
	OrgID     uint
	Filter    string
}

func (e ChartNotFoundError) Error() string {
	return fmt.Sprintf("chart not found. OrgID: %d, ChartInfo: %s", e.OrgID, e.ChartInfo)
}

func (e ChartNotFoundError) Details() []interface{} {
	return []interface{}{"organizationId", e.OrgID, "chartInfo", e.ChartInfo}
}

func (ChartNotFoundError) ServiceError() bool {
	return true
}

func (ChartNotFoundError) NotFound() bool {
	return true
}
