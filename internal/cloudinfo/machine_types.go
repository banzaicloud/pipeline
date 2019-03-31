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
	"context"
	"sync"

	"github.com/banzaicloud/pipeline/.gen/cloudinfo"
	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type VMKey struct {
	cloud        string
	service      string
	region       string
	instanceType string
}

type InstanceTypeMap struct {
	lock     sync.RWMutex
	internal map[VMKey]cloudinfo.ProductDetails
}

func NewInstanceTypeMap() *InstanceTypeMap {
	return &InstanceTypeMap{
		internal: make(map[VMKey]cloudinfo.ProductDetails),
	}
}

func (im *InstanceTypeMap) getMachine(key VMKey) (cloudinfo.ProductDetails, bool) {
	im.lock.RLock()
	result, ok := im.internal[key]
	im.lock.RUnlock()
	return result, ok
}

func (im *InstanceTypeMap) setMachines(cloud string, service string, region string, vmList []cloudinfo.ProductDetails) {
	instanceTypeMap.lock.Lock()
	for _, product := range vmList {
		instanceTypeMap.internal[VMKey{
			cloud,
			service,
			region,
			product.Type,
		}] = product
	}
	instanceTypeMap.lock.Unlock()
}

// nolint: gochecknoglobals
var instanceTypeMap = NewInstanceTypeMap()

func fetchMachineTypes(logger logrus.FieldLogger, cloud string, service string, region string) error {
	cloudInfoEndPoint := viper.GetString(config.CloudInfoEndPoint)
	if len(cloudInfoEndPoint) == 0 {
		return emperror.With(errors.New("missing config"), "cloudInfoEndPoint", config.CloudInfoEndPoint)
	}

	log := logger.WithFields(logrus.Fields{"cloudInfoEndPoint": cloudInfoEndPoint, "cloud": cloud, "region": region, "service": service})
	log.Info("fetching machine types from CloudInfo")

	cloudInfoClient := cloudinfo.NewAPIClient(&cloudinfo.Configuration{
		BasePath:      cloudInfoEndPoint,
		DefaultHeader: make(map[string]string),
		UserAgent:     "Pipeline/go",
	})
	response, _, err := cloudInfoClient.ProductsApi.GetProducts(context.Background(), cloud, service, region)
	if err != nil {
		return errors.WithStack(err)
	}

	instanceTypeMap.setMachines(cloud, service, region, response.Products)
	return nil
}

//GetMachineDetails returns machine resource details, like cpu/gpu/memory etc. either from local cache or CloudInfo
func GetMachineDetails(logger logrus.FieldLogger, cloud string, service string, region string, instanceType string) (*cloudinfo.ProductDetails, error) {

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
			return nil, emperror.WrapWith(err, "failed to retrieve machine types", "cloud", cloud, "region", region, "service", service)
		}
		vmDetails, ok = instanceTypeMap.getMachine(vmKey)
		if !ok {
			return nil, emperror.WrapWith(err, "no machine info found for VM instance", "cloud", cloud, "region", region, "service", service, "instanceType", instanceType)
		}
	}

	return &vmDetails, nil
}
