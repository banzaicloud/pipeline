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
	"testing"

	"gopkg.in/yaml.v2"
)

func TestUnmarshalCICDRepoConfig(t *testing.T) {
	configYaml := `
cluster:
  name: "banzai-cicd-cluster"
  provider: "google"
workspace:
  base: /go
  path: src/github.com/banzaicloud/pipeline
pipeline:
  print_env:
    image: golang:1.10
    commands:
    - pwd
    - env
    - find .
    group: build
  build:
    image: golang:1.10
    commands:
    - make build
  test:
    image: golang:1.10
    commands:
    - mkdir $HOME/config
    - cp config/config.yaml.dist $HOME/config/config.yaml
    - make test
    environment:
      VAULT_ADDR: http://vault:8200
      VAULT_TOKEN: 227e1cce-6bf7-30bb-2d2a-acc854318caf
  build_container:
    image: plugins/docker
    dockerfile: Dockerfile
    repo: banzaicloud/pipeline
    tags: "{{ printf \"%s\" .CICD_BRANCH }}"
    log: debug
services:
  vault:
    image: vault:0.10.4
    ports:
    - 8200
    environment:
      SKIP_SETCAP: "true"
      VAULT_DEV_ROOT_TOKEN_ID: 227e1cce-6bf7-30bb-2d2a-acc854318caf
`

	config := cicdRepoConfig{}
	err := yaml.Unmarshal([]byte(configYaml), &config)

	if err != nil {
		t.Error("Unmarshal expected to succeed but got error: ", err.Error())
	}

	_, err = yaml.Marshal(config)

	if err != nil {
		t.Error("Marshal expected to succeed but got error: ", err.Error())
	}

}
