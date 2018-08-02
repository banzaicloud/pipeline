package eks

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/banzaicloud/pipeline/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	eksVPCTemplateName      = "amazon-eks-vpc-cf.yaml"
	eksNodePoolTemplateName = "amazon-eks-nodepool-cf.yaml"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

// getEksCloudFormationTemplate returns CloudFormation template with given name
func getEksCloudFormationTemplate(name string) (string, error) {

	// location to retrieve the Cloud Formation template from
	templatePath := viper.GetString(config.EksTemplateLocation) + "/" + name

	log.Infof("Getting CloudFormation template from %q", templatePath)

	u, err := url.Parse(templatePath)
	if err != nil {
		log.Errorf("Getting CloudFormation template from %q failed: %s", templatePath, err.Error())
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
		err = fmt.Errorf("Not supported scheme: %s", u.Scheme)
	}

	if err != nil {
		log.Errorf("Reading CloudFormation template content from %q failed: %s", templatePath, err.Error())
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
