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
  create_cluster:
    action: EnsureCluster
    cluster:
      cloud: google
      location: europe-west1-b
      name: banzaicloudleu1
      postHooks:
        InstallLogging:
          bucketName: johndoe-spark-cluster-logs
          secretId: d316a5c2c3a107aa47f0bda16cdc70020895a01d5d23847014013780bb52a7d9
        InstallMonitoring: {}
      profileName: ""
      properties:
        gke:
          master:
            version: "1.10"
          nodePools:
            pool1:
              autoscaling: true
              count: 2
              instanceType: n1-standard-1
              maxCount: 3
              minCount: 2
            system:
              autoscaling: true
              count: 1
              instanceType: n1-standard-2
              maxCount: 2
              minCount: 1
          nodeVersion: "1.10"
      secretId: c8f9c9fc3835b9a3721165afea97ffb78e1375552ab112ed54aee30b29c962ae
      secretName: ""
    cluster_secret:
      name: null
      namespace: '{{ .DRONE_NAMESPACE }}'
      spec:
      - name: DOCKER_USERNAME
        source: username
      - name: DOCKER_PASSWORD
        source: password
    image: banzaicloud/ci-pipeline-client:latest
{{{{ if .debugMode }}}}
  env:
    image: node:10
    commands:
    - env
    - find .
{{{{ end }}}}
{{{{ if .runTests }}}}
  test:
    image: node:10
    commands:
    - npm version
{{{{ end }}}}
  build_container:
    image: plugins/docker
    dockerfile: Dockerfile
    repo: '{{ .DRONE_REPO }}'
    tags: '{{ trunc 7 .DRONE_COMMIT_SHA }}'
  package_application:
    when:
      branch:
        include:
        - master
    image: lachlanevenson/k8s-helm:latest
    commands:
    - helm init -c
    - helm repo add banzaicloud-stable http://kubernetes-charts.banzaicloud.com/branch/master
    - helm package -u ./.banzaicloud/charts/spotguide-nodejs-mongodb
  install_application_secret:
    action: InstallSecret
    cluster_secret:
      name: ""
      namespace: default
      spec:
      - name: mongodb-username
        source: username
      - name: mongodb-password
        source: password
      - name: mongodb-root-password
        source: password
    image: banzaicloud/ci-pipeline-client:latest
    when:
      branch:
        include:
        - master
  deploy_application:
    action: EnsureDeployment
    deployment:
      name: ./spotguide-nodejs-mongodb-1.0.0.tgz
      releaseName: '{{ .DRONE_REPO_NAME }}'
      reuseValues: true
      values:
        application:
          deployment:
            image:
              pullPolicy: Always
              repository: '{{ .DRONE_REPO }}'
              tag: '{{ trunc 7 .DRONE_COMMIT_SHA }}'
          ingress:
            hosts:
            - app.{{ .DRONE_REPO_NAME }}.{{ .CLUSTER_NAME }}.{{ .ORG_NAME }}.banzaicloud.io
        mongodb:
          existingSecret:
          mongodbDatabase: null
          mongodbUsername: null
    image: banzaicloud/ci-pipeline-client:latest
    when:
      branch:
        include:
        - master
`

const testExpectedPipelineYAML = `
pipeline:
  create_cluster:
    action: EnsureCluster
    cluster:
      cloud: google
      location: europe-west2-a
      name: banzaicloudsgts
      postHooks:
        InstallLogging:
          bucketName: johndoe-spark-cluster-logs
          secretId: b6d88b1c21908689d80f4c5a0c32d86666e1bfd90e14602d1fd6eccd6c232281
        InstallMonitoring: {}
      profileName: ""
      properties:
        gke:
          master:
            version: "1.10"
          nodePools:
            pool1:
              autoscaling: true
              count: 2
              instanceType: n1-standard-1
              maxCount: 3
              minCount: 2
            system:
              autoscaling: true
              count: 1
              instanceType: n1-standard-2
              maxCount: 2
              minCount: 1
          nodeVersion: "1.10"
          projectId: ""
      secretId: c8f9c9fc3835b9a3721165afea97ffb78e1375552ab112ed54aee30b29c962ae
      secretName: ""
    cluster_secret:
      name: spotguide-nodejs-mongodb-05-docker-hub
      namespace: '{{ .DRONE_NAMESPACE }}'
      spec:
      - name: DOCKER_USERNAME
        source: username
      - name: DOCKER_PASSWORD
        source: password
    image: banzaicloud/ci-pipeline-client:latest
  test:
    image: node:10
    commands:
    - npm version
  build_container:
    image: plugins/docker
    dockerfile: Dockerfile
    repo: '{{ .DRONE_REPO }}'
    tags: '{{ trunc 7 .DRONE_COMMIT_SHA }}'
  package_application:
    when:
      branch:
        include:
        - master
    image: lachlanevenson/k8s-helm:latest
    commands:
    - helm init -c
    - helm repo add banzaicloud-stable http://kubernetes-charts.banzaicloud.com/branch/master
    - helm package -u ./.banzaicloud/charts/spotguide-nodejs-mongodb
  install_application_secret:
    action: InstallSecret
    cluster_secret:
      name: spotguide-nodejs-mongodb-05-mongodb
      namespace: default
      spec:
      - name: mongodb-username
        source: username
      - name: mongodb-password
        source: password
      - name: mongodb-root-password
        source: password
    image: banzaicloud/ci-pipeline-client:latest
    when:
      branch:
        include:
        - master
  deploy_application:
    action: EnsureDeployment
    deployment:
      name: ./spotguide-nodejs-mongodb-1.0.0.tgz
      releaseName: '{{ .DRONE_REPO_NAME }}'
      reuseValues: true
      values:
        application:
          deployment:
            image:
              pullPolicy: Always
              repository: '{{ .DRONE_REPO }}'
              tag: '{{ trunc 7 .DRONE_COMMIT_SHA }}'
          ingress:
            hosts:
            - app.{{ .DRONE_REPO_NAME }}.{{ .CLUSTER_NAME }}.{{ .ORG_NAME }}.banzaicloud.io
        mongodb:
          existingSecret: spotguide-nodejs-mongodb-05-mongodb
          mongodbDatabase: application
          mongodbUsername: user
    image: banzaicloud/ci-pipeline-client:latest
    when:
      branch:
        include:
        - master
`

var testLaunchRequestJSON = `{
  "spotguideName": "banzaicloud/spotguide-nodejs-mongodb",
  "repoPrivate": true,
  "spotguideVersion": "v0.0.3",
  "repoOrganization": "banzaicloud",
  "repoName": "spotguide-nodejs-mongodb-05",
  "cluster": {
      "name": "banzaicloudsgts",
      "location": "europe-west2-a",
      "cloud": "google",
      "secretId": "c8f9c9fc3835b9a3721165afea97ffb78e1375552ab112ed54aee30b29c962ae",
      "properties": {
          "gke": {
              "master": {
                  "version": "1.10"
              },
              "nodeVersion": "1.10",
              "nodePools": {
                  "pool1": {
                      "count": 2,
                      "autoscaling": true,
                      "instanceType": "n1-standard-1",
                      "minCount": 2,
                      "maxCount": 3
                  },
                  "system": {
                      "count": 1,
                      "autoscaling": true,
                      "instanceType": "n1-standard-2",
                      "minCount": 1,
                      "maxCount": 2
                  }
              }
          }
      },
      "postHooks": {
          "InstallLogging": {
              "secretId": "b6d88b1c21908689d80f4c5a0c32d86666e1bfd90e14602d1fd6eccd6c232281",
              "bucketName": "johndoe-spark-cluster-logs"
          },
          "InstallMonitoring": {}
      }
  },
  "secrets": [
      {
          "name": "spotguide-nodejs-mongodb-05-mongodb",
          "type": "password",
          "values": {
              "username": "user",
              "password": null
          }
      },
      {
          "name": "spotguide-nodejs-mongodb-05-docker-hub",
          "type": "password",
          "values": {
              "username": "johndoe",
              "password": "mys3cret"
          }
      }
  ],
  "pipeline": {
      "deploy_application": {
          "deployment": {
              "values": {
                  "mongodb": {
                      "mongodbDatabase": "application",
                      "existingSecret": "spotguide-nodejs-mongodb-05-mongodb",
                      "mongodbUsername": "user"
                  }
              }
          }
      },
      "install_application_secret": {
          "cluster_secret": {
              "name": "spotguide-nodejs-mongodb-05-mongodb"
          }
      },
      "create_cluster": {
          "cluster_secret": {
              "name": "spotguide-nodejs-mongodb-05-docker-hub"
          }
      }
  },
  "prePipeline": {
    "debugMode": false,
    "runTests": true
  }
}`

func TestDroneRepoConfigPipeline(t *testing.T) {

	launchRequest := LaunchRequest{}
	err := json.Unmarshal([]byte(testLaunchRequestJSON), &launchRequest)

	if err != nil {
		t.Fatal("Unmarshal expected to succeed but got error: ", err.Error())
	}

	droneConfig, err := createDroneRepoConfig([]byte(testPipelineYAML), &launchRequest)

	if err != nil {
		t.Fatal("createDroneRepoConfig expected to succeed but got error: ", err.Error())
	}

	actualPipelineYAML, err := yaml.Marshal(&droneConfig)

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
