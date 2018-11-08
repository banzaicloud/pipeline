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

type PipelineMetrics struct {
	clusterStatus *prometheus.GaugeVec

	metricsMtx sync.RWMutex
	sync.RWMutex
}

type scrapeResultTotalCluster struct {
	provider string
	location string
	status   string
}

var (
	StatusChangeDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "cluster",
		Name:      "status_change_duration",
		Help:      "Cluster status change duration in seconds",
	},
		[]string{"provider", "location", "status"},
	)
)

func NewExporter() *PipelineMetrics {
	p := PipelineMetrics{
		clusterStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "cluster",
			Name:      "total",
			Help:      "Total number of clusters, partitioned by provider, location and status",
		},
			[]string{"provider", "location", "status"},
		),
	}
	return &p
}

func (p *PipelineMetrics) Describe(ch chan<- *prometheus.Desc) {
	p.clusterStatus.Describe(ch)
}

func (p *PipelineMetrics) Collect(ch chan<- prometheus.Metric) {
	clusterTotal := make(chan scrapeResultTotalCluster)

	p.Lock()
	defer p.Unlock()

	go p.scrape(clusterTotal)
	p.setClusterMetrics(clusterTotal)

	p.clusterStatus.Collect(ch)
}

func (p *PipelineMetrics) scrape(scrapesTotalCluster chan<- scrapeResultTotalCluster) {

	defer close(scrapesTotalCluster)
	clusters := intCluster.NewClusters(config.DB())
	allCluster, err := clusters.All()
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

func (p *PipelineMetrics) setClusterMetrics(resultTotalCluster <-chan scrapeResultTotalCluster) {
	log.Debug("set cluster metrics")
	p.metricsMtx.Lock()
	p.clusterStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "cluster",
		Name:      "total",
		Help:      "Total number of clusters, partitioned by provider, location and status",
	},
		[]string{"provider", "location", "status"})
	p.metricsMtx.Unlock()

	for scr := range resultTotalCluster {
		var labels prometheus.Labels = map[string]string{
			"provider": scr.provider,
			"location": scr.location,
			"status":   scr.status,
		}
		p.clusterStatus.With(labels).Inc()
	}
}
