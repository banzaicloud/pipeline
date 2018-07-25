package eks

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/hashicorp/go-getter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
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

	stateStorePath := config.GetStateStorePath("")
	dir, err := ioutil.TempDir(stateStorePath, "eks-templates")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(dir)

	// path to save the CloudFormation template to
	templatePath := filepath.Join(dir, "cf-template.yaml")

	// location to retrieve the Cloud Formation template from
	templateSrcLocation := filepath.Join(viper.GetString(config.EksTemplateLocation), name)

	log.Infof("Getting CloudFormation template from %q to %q", templateSrcLocation, templatePath)

	err = getter.GetFile(templatePath, templateSrcLocation)
	if err != nil {
		log.Errorf("Getting CloudFormation template from %q failed: %s", templateSrcLocation, err.Error())
		return "", err
	}

	content, err := ioutil.ReadFile(templatePath)
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
