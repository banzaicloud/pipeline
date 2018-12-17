// Copyright © 2018 Banzai Cloud
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
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

var globalCache *cache.Cache

func init() {
	globalCache = cache.New(1*time.Minute, 5*time.Minute)
}

type releaseCache interface {
	// GetReleases returns the cached release list.
	GetReleases(clusterKey string) (*rls.ListReleasesResponse, bool)

	// SaveReleases fills the cache.
	SaveReleases(clusterKey string, releases *rls.ListReleasesResponse)

	// ClearReleases clears the release list cache for a cluster.
	ClearReleases(clusterKey string)
}

type goCacheReleaseCache struct {
	cache  *cache.Cache
	logger logrus.FieldLogger
}

func newGoCacheReleaseCache(logger logrus.FieldLogger) *goCacheReleaseCache {
	return &goCacheReleaseCache{
		cache:  globalCache,
		logger: logger,
	}
}

func (c *goCacheReleaseCache) GetReleases(clusterKey string) (*rls.ListReleasesResponse, bool) {
	result, ok := c.cache.Get(clusterKey)
	if !ok {
		c.logger.WithField("clusterKey", clusterKey).Debug("cache miss for helm releases")

		return nil, false
	}

	releases, ok := result.(*rls.ListReleasesResponse)

	if ok {
		c.logger.WithField("clusterKey", clusterKey).Debug("cache hit for helm releases")
	}

	return releases, ok
}

func (c *goCacheReleaseCache) SaveReleases(clusterKey string, releases *rls.ListReleasesResponse) {
	c.logger.WithField("clusterKey", clusterKey).Debug("saving helm release cache")

	c.cache.SetDefault(clusterKey, releases)
}

func (c *goCacheReleaseCache) ClearReleases(clusterKey string) {
	c.logger.WithField("clusterKey", clusterKey).Debug("cleaning helm release cache")

	c.cache.Delete(clusterKey)
}
