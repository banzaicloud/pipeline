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

package global

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/spf13/viper"
)

type AwsTag string

const ManagedByPipelineTag = "banzaicloud-pipeline-uuid"

// PipelineUUID returns an UUID that identifies the specific installation (deployment) of the platform.
// If UUID is not available, empty string is returned instead of generating a random one, because no UUID is better than one that always changes.
func PipelineUUID() string {
	return viper.GetString(config.PipelineUUID)
}
