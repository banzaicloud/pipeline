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

package services

// ServiceNameMapper maps integrated service names that are different in external / legacy systems
type ServiceNameMapper interface {
	MapServiceName(serviceName string) string
}

type svcNameMapper struct {
	mappings map[string]string
}

func NewServiceNameMapper() ServiceNameMapper {
	return svcNameMapper{
		map[string]string{
			"dns":          "external-dns",
			"external-dns": "dns",
		},
	}
}

func (s svcNameMapper) MapServiceName(serviceName string) string {
	if mapped, ok := s.mappings[serviceName]; ok {
		return mapped
	}

	return serviceName
}
