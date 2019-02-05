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

package eks

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/banzaicloud/pipeline/config"
	"github.com/spf13/viper"
)

const (
	eksVPCTemplateName      = "amazon-eks-vpc-cf.yaml"
	eksNodePoolTemplateName = "amazon-eks-nodepool-cf.yaml"
)

type CFTemplate struct {
	Content string
	Version string
}

const Version0_1 = "0.1"
const Version0_2 = "0.2"
const Version1_0 = "1.0"

const MajorVersionSeparator = "."

func (c *CFTemplate) MinorVersionChange(version *CFTemplate) bool {
	version1A := strings.Split(c.Version, MajorVersionSeparator)
	version2A := strings.Split(version.Version, MajorVersionSeparator)
	return version1A[0] == version2A[0]
}

// getEksCloudFormationTemplate returns CloudFormation template with given name
func getEksCloudFormationTemplate(name string) (*CFTemplate, error) {
	cfTemplate := CFTemplate{}

	// location to retrieve the Cloud Formation template from
	cfTemplate.Version = viper.GetString(config.EksTemplateVersion)
	templatePath := path.Join(viper.GetString(config.EksTemplateLocation), cfTemplate.Version, name)

	log.Infof("Getting CloudFormation template from %q", templatePath)

	u, err := url.Parse(templatePath)
	if err != nil {
		log.Errorf("Getting CloudFormation template from %q failed: %s", templatePath, err.Error())
		return &cfTemplate, err
	}

	var content []byte
	if u.Scheme == "file" || u.Scheme == "" {
		content, err = ioutil.ReadFile(templatePath)
	} else if u.Scheme == "http" || u.Scheme == "https" {
		var resp *http.Response
		resp, err = http.Get(u.String())
		if err == nil {
			content, err = ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
		}
	} else {
		err = fmt.Errorf("Not supported scheme: %s", u.Scheme)
	}

	if err != nil {
		log.Errorf("Reading CloudFormation template content from %q failed: %s", templatePath, err.Error())
		return &cfTemplate, err
	}
	cfTemplate.Content = string(content)

	return &cfTemplate, nil
}

// GetVPCTemplate returns the CloudFormation template for creating VPC for EKS cluster
func GetVPCTemplate() (*CFTemplate, error) {
	return getEksCloudFormationTemplate(eksVPCTemplateName)
}

// GetNodePoolTemplate returns the CloudFormation template for creating node pools for EKS cluster
func GetNodePoolTemplate() (*CFTemplate, error) {
	return getEksCloudFormationTemplate(eksNodePoolTemplateName)
}
