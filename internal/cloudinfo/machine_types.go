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

package cloudinfo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"

	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
)

type CloudInfoResponse struct {

	// Products represents a slice of products for a given provider (VMs with attributes and process)
	Products []*MachineDetails `json:"products"`

	// ScrapingTime represents scraping time for a given provider in milliseconds
	ScrapingTime string `json:"scrapingTime,omitempty"`
}

// SpotPriceInfo represents different prices per availability zones
type SpotPriceInfo map[string]float64

type MachineDetails struct {
	Type          string            `json:"type"`
	OnDemandPrice float64           `json:"onDemandPrice"`
	SpotPrice     SpotPriceInfo     `json:"spotPrice"`
	Cpus          float64           `json:"cpusPerVm"`
	Mem           float64           `json:"memPerVm"`
	Gpus          float64           `json:"gpusPerVm"`
	NtwPerf       string            `json:"ntwPerf"`
	NtwPerfCat    string            `json:"ntwPerfCategory"`
	Zones         []string          `json:"zones"`
	Attributes    map[string]string `json:"attributes"`
}

type VMKey struct {
	cloud        string
	service      pkgCluster.DistributionID
	region       string
	instanceType string
}

// nolint: gochecknoglobals
var instanceTypeMap = make(map[VMKey]MachineDetails)

func fetchMachineTypes(cloud string, service pkgCluster.DistributionID, region string) error {
	cloudInfoEndPoint := viper.GetString(config.CloudInfoEndPoint)
	if len(cloudInfoEndPoint) == 0 {
		return emperror.With(errors.New("missing config"), "propertyName", config.CloudInfoEndPoint)
	}
	cloudInfoUrl := fmt.Sprintf(
		"%s/providers/%s/services/%s/regions/%s/products",
		cloudInfoEndPoint, cloud, service, region)
	ciRequest, err := http.NewRequest(http.MethodGet, cloudInfoUrl, nil)
	if err != nil {
		return errors.New("error fetching machine types from CloudInfo")
	}

	ciRequest.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}

	ciResponse, err := httpClient.Do(ciRequest)
	if err != nil {
		return errors.New("error fetching machine types from CloudInfo")
	}
	respBody, _ := ioutil.ReadAll(ciResponse.Body)
	var vmDetails CloudInfoResponse
	json.Unmarshal(respBody, &vmDetails)

	for _, product := range vmDetails.Products {
		instanceTypeMap[VMKey{
			cloud,
			service,
			region,
			product.Type,
		}] = *product
	}

	return nil
}

//GetMachineDetails returns machine resource details, like cpu/gpu/memory etc. either from local cache or CloudInfo
func GetMachineDetails(cloud string, service pkgCluster.DistributionID, region string, instanceType string) (*MachineDetails, error) {

	vmKey := VMKey{
		cloud,
		service,
		region,
		instanceType,
	}

	vmDetails, ok := instanceTypeMap[vmKey]
	if !ok {
		err := fetchMachineTypes(cloud, service, region)
		if err != nil {
			return nil, err
		}
		vmDetails = instanceTypeMap[vmKey]
	}

	return &vmDetails, nil
}
