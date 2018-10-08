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

package spotguide

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const testPipelineYAML = `
pipeline:
  deploy_application:
    image: banzaicloud/ci-pipeline-client:latest
    action: EnsureDeployment
    deployment:
      name: ./spotguide-nodejs-mongodb-1.0.0.tgz
      reuseValues: true
      releaseName: '{{ .DRONE_REPO_NAME }}'
      values:
        deployment:
          repository: '{{ .DRONE_REPO }}'
          tag: '{{ trunc 7 .DRONE_COMMIT_SHA }}'
          pullPolicy: Always
`

const testExpectedPipelineYAML = `
pipeline:
  deploy_application:
    image: banzaicloud/ci-pipeline-client:latest
    action: EnsureDeployment
    deployment:
      name: ./spotguide-nodejs-mongodb-1.0.0.tgz
      reuseValues: true
      releaseName: '{{ .DRONE_REPO_NAME }}'
      values:
        application:
          key: value
        deployment:
          repository: '{{ .DRONE_REPO }}'
          tag: '{{ trunc 7 .DRONE_COMMIT_SHA }}'
          pullPolicy: Always
`

var testLaunchRequestJSON = `{
	"pipeline": {
		"deploy_application": {
			"deployment": {
				"values": {
					"application": {
					    "key": "value"
					}
				}
			}
		}
	}
}`

func TestDroneRepoConfigPipeline(t *testing.T) {

	config := droneRepoConfig{}
	err := yaml.Unmarshal([]byte(testPipelineYAML), &config)

	if err != nil {
		t.Fatal("Unmarshal expected to succeed but got error: ", err.Error())
	}

	launchRequest := LaunchRequest{}
	err = json.Unmarshal([]byte(testLaunchRequestJSON), &launchRequest)

	if err != nil {
		t.Fatal("Unmarshal expected to succeed but got error: ", err.Error())
	}

	err = droneRepoConfigPipeline(&launchRequest, &config)

	if err != nil {
		t.Fatal("droneRepoConfigPipeline expected to succeed but got error: ", err.Error())
	}

	actualPipelineYAML, err := yaml.Marshal(&config)

	if err != nil {
		t.Error("Marshal expected to succeed but got error: ", err.Error())
	}

	// Save this for debugging purposes
	// err = ioutil.WriteFile("./actualPipeline.yaml", actualPipelineYAML, 0644)
	// if err != nil {
	// 	t.Fatal("WriteFile expected to succeed but got error: ", err.Error())
	// }

	expectedConfig := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(testExpectedPipelineYAML), &expectedConfig)

	if err != nil {
		t.Fatal("Unmarshal expected to succeed but got error: ", err.Error())
	}

	actualConfig := map[string]interface{}{}
	err = yaml.Unmarshal(actualPipelineYAML, &actualConfig)

	if err != nil {
		t.Fatal("Unmarshal expected to succeed but got error: ", err.Error())
	}

	assert.Equal(t, expectedConfig, actualConfig, "Actual pipeline.yaml doesn't match the expected one")
}
