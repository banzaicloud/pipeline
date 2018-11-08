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

package cluster

import (
	"sync"

	"github.com/banzaicloud/pipeline/config"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/prometheus/client_golang/prometheus"
)

type pipelineMetrics struct {
	clusterStatus *prometheus.GaugeVec
	clusters      *intCluster.Clusters

	mu sync.Mutex
}

type scrapeResultTotalCluster struct {
	provider string
	location string
	status   string
}

var (
	StatusChangeDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "pipeline",
		Name:      "status_change_duration",
		Help:      "Cluster status change duration in seconds",
	},
		[]string{"provider", "location", "status"},
	)
)

func NewExporter() *pipelineMetrics {
	p := pipelineMetrics{
		clusterStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pipeline",
			Name:      "total",
			Help:      "Total number of clusters, partitioned by provider, location and status",
		},
			[]string{"provider", "location", "status"},
		),
		clusters: intCluster.NewClusters(config.DB()),
	}
	return &p
}

func (p *pipelineMetrics) Describe(ch chan<- *prometheus.Desc) {
	p.clusterStatus.Describe(ch)
}

func (p *pipelineMetrics) Collect(ch chan<- prometheus.Metric) {
	clusterTotal := make(chan scrapeResultTotalCluster)

	p.mu.Lock()
	defer p.mu.Unlock()

	go p.scrape(clusterTotal)
	p.setClusterMetrics(clusterTotal)

	p.clusterStatus.Collect(ch)
}

func (p *pipelineMetrics) scrape(scrapesTotalCluster chan<- scrapeResultTotalCluster) {

	defer close(scrapesTotalCluster)

	allCluster, err := p.clusters.All()
	if err != nil {
		return
	}

	for _, cluster := range allCluster {
		scrapesTotalCluster <- scrapeResultTotalCluster{
			provider: cluster.Cloud,
			location: cluster.Location,
			status:   cluster.Status,
		}
	}

}

func (p *pipelineMetrics) setClusterMetrics(resultTotalCluster <-chan scrapeResultTotalCluster) {
	log.Debug("set cluster metrics")
	p.clusterStatus.Reset()

	for scr := range resultTotalCluster {
		var labels prometheus.Labels = map[string]string{
			"provider": scr.provider,
			"location": scr.location,
			"status":   scr.status,
		}
		p.clusterStatus.With(labels).Inc()
	}
}
