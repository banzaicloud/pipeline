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
	"fmt"
	"strings"
)

// decouple helm lib types from the api
type ChartDetails = map[string]interface{}

// decouple helm lib types from the api
type ChartList = []interface{}

// ChartFilter filter data for chart retrieval
type ChartFilter struct {
	Repo    []string
	Name    []string
	Version []string
	Keyword []string
}

func (cf ChartFilter) String() string {
	return fmt.Sprintf("repo: %s, chart: %s, version %s", cf.RepoFilter(), cf.NameFilter(), cf.VersionFilter())
}

// RepoFilter gets the string filter
func (cf ChartFilter) RepoFilter() string {

	repoFilter := firstOrEmpty(cf.Repo)

	// trim  regexp markers -if exist
	return strings.TrimSuffix(strings.TrimPrefix(repoFilter, "^"), "$")
}

// StrictRepoFilter wraps the filter with regexp markers for exact match
func (cf ChartFilter) StrictRepoFilter() string {
	repoFilter := cf.RepoFilter()
	if repoFilter != "" {
		repoFilter = fmt.Sprintf("%s%s%s", "^", repoFilter, "$")
	}
	return repoFilter
}

func (cf ChartFilter) NameFilter() string {
	// exact match always!
	chartNameFilter := firstOrEmpty(cf.Name)
	if chartNameFilter != "" {
		chartNameFilter = fmt.Sprintf("%s%s%s", "^", chartNameFilter, "$")
	}
	return chartNameFilter
}

func (cf ChartFilter) VersionFilter() string {
	versionFilter := firstOrEmpty(cf.Version)
	// special case (backwards comp.)
	if versionFilter == "all" {
		return versionFilter
	}
	if versionFilter != "" {
		versionFilter = fmt.Sprintf("%s%s", "^", versionFilter)
	}

	return versionFilter
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
