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
	"sync"

	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
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
	service      string
	region       string
	instanceType string
}

type InstanceTypeMap struct {
	lock     sync.RWMutex
	internal map[VMKey]MachineDetails
}

func NewInstanceTypeMap() *InstanceTypeMap {
	return &InstanceTypeMap{
		internal: make(map[VMKey]MachineDetails),
	}
}

func (im *InstanceTypeMap) getMachine(key VMKey) (MachineDetails, bool) {
	im.lock.RLock()
	result, ok := im.internal[key]
	im.lock.RUnlock()
	return result, ok
}

func (im *InstanceTypeMap) setMachines(cloud string, service string, region string, vmList []*MachineDetails) {
	instanceTypeMap.lock.Lock()
	for _, product := range vmList {
		instanceTypeMap.internal[VMKey{
			cloud,
			service,
			region,
			product.Type,
		}] = *product
	}
	instanceTypeMap.lock.Unlock()
}

// nolint: gochecknoglobals
var instanceTypeMap = NewInstanceTypeMap()

func fetchMachineTypes(logger logrus.FieldLogger, cloud string, service string, region string) error {
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

	log := logger.WithFields(logrus.Fields{"cloudInfoEndPoint": cloudInfoEndPoint, "cloud": cloud, "region": region, "service": service})
	log.Info("fetching machine types from CloudInfo")

	ciResponse, err := httpClient.Do(ciRequest)
	if err != nil {
		return emperror.Wrap(err, "error fetching machine types from CloudInfo")
	}
	respBody, _ := ioutil.ReadAll(ciResponse.Body)
	var vmDetails CloudInfoResponse
	json.Unmarshal(respBody, &vmDetails)

	instanceTypeMap.setMachines(cloud, service, region, vmDetails.Products)

	return nil
}

//GetMachineDetails returns machine resource details, like cpu/gpu/memory etc. either from local cache or CloudInfo
func GetMachineDetails(logger logrus.FieldLogger, cloud string, service string, region string, instanceType string) (*MachineDetails, error) {

	vmKey := VMKey{
		cloud,
		service,
		region,
		instanceType,
	}

	vmDetails, ok := instanceTypeMap.getMachine(vmKey)
	if !ok {
		err := fetchMachineTypes(logger, cloud, service, region)
		if err != nil {
			return nil, emperror.WrapWith(err, "failed to retrieve service machine types", "cloud", cloud, "region", region, "service", service)
		}
		vmDetails, ok = instanceTypeMap.getMachine(vmKey)
		if !ok {
			return nil, emperror.WrapWith(err, "no machine info found for VM instance", "cloud", cloud, "region", region, "service", service, "instanceType", instanceType)
		}
	}

	return &vmDetails, nil
}
