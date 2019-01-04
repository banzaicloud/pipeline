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

package acsk

const (
	AlibabaClusterStateRunning      = "running"
	AlibabaClusterStateFailed       = "failed"
	DefaultMasterInstanceType       = "ecs.sn1.large"
	DefaultMasterSystemDiskCategory = "cloud_efficiency"
	DefaultMasterSystemDiskSize     = 40
	DefaultWorkerInstanceType       = "ecs.sn1.large"
	AlibabaStartCreateClusterLog    = "start to createk8scluster"
	AlibabaCreateClusterFailedLog   = "start to update cluster status create_failed"
	AlibabaStartScaleClusterLog     = "start to scale kubernetes cluster"
	AlibabaScaleClusterFailedLog    = "start to update cluster status update_failed"
	AlibabaApiDomain                = "cs.aliyuncs.com"
	AlibabaInstanceHealthyStatus    = "Healthy"
)
