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

	"github.com/banzaicloud/pipeline/config"
	"github.com/spf13/viper"
)

const (
	eksVPCTemplateName      = "amazon-eks-vpc-cf.yaml"
	eksSubnetTemplateName   = "amazon-eks-subnet-cf.yaml"
	eksNodePoolTemplateName = "amazon-eks-nodepool-cf.yaml"
)

// getEksCloudFormationTemplate returns CloudFormation template with given name
func getEksCloudFormationTemplate(name string) (string, error) {

	// location to retrieve the Cloud Formation template from
	templatePath := viper.GetString(config.EksTemplateLocation) + "/" + name

	log.Infof("getting CloudFormation template from %q", templatePath)

	u, err := url.Parse(templatePath)
	if err != nil {
		log.Errorf("getting CloudFormation template from %q failed: %s", templatePath, err.Error())
		return "", err
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
		err = fmt.Errorf("not supported scheme: %s", u.Scheme)
	}

	if err != nil {
		log.Errorf("reading CloudFormation template content from %q failed: %s", templatePath, err.Error())
		return "", err
	}

	return string(content), nil
}

// GetVPCTemplate returns the CloudFormation template for creating VPC for EKS cluster
func GetVPCTemplate() (string, error) {
	return getEksCloudFormationTemplate(eksVPCTemplateName)
}

// GetNodePoolTemplate returns the CloudFormation template for creating node pools for EKS cluster
func GetNodePoolTemplate() (string, error) {
	return getEksCloudFormationTemplate(eksNodePoolTemplateName)
}

// GetSubnetTemplate returns the CloudFormation template for creating a Subnet
func GetSubnetTemplate() (string, error) {
	return getEksCloudFormationTemplate(eksSubnetTemplateName)
}
