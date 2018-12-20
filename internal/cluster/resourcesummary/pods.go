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

package resourcesummary

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	resourcev1 "k8s.io/kubernetes/pkg/api/v1/resource"
)

// CalculatePodsTotalRequestsAndLimits calculates requests and limits of all the given pods.
func CalculatePodsTotalRequestsAndLimits(pods []v1.Pod) (map[v1.ResourceName]resource.Quantity, map[v1.ResourceName]resource.Quantity) {
	reqs, limits := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, pod := range pods {
		podReqs, podLimits := resourcev1.PodRequestsAndLimits(&pod)

		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = *podReqValue.Copy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}

		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = *podLimitValue.Copy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}

	return reqs, limits
}
