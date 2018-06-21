package verify

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/banzaicloud/pipeline/config"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/sirupsen/logrus"
)

const (
	defaultRegion = "eu-west-1"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

// awsVerify for validation AWS credentials
type awsVerify struct {
	credentials *credentials.Credentials
}

// CreateAWSSecret create a new 'awsVerify' instance
func CreateAWSSecret(values map[string]string) *awsVerify {
	return &awsVerify{
		credentials: CreateAWSCredentials(values),
	}
}

// VerifySecret validates AKS credentials
func (a *awsVerify) VerifySecret() error {
	client, err := CreateEC2Client(a.credentials, defaultRegion)
	if err != nil {
		return err
	}

	// currently the only way to verify AWS credentials is to actually use them to sign a request and see if it works
	_, err = client.DescribeRegions(nil)
	return err
}

// CreateEC2Client create a new ec2 instance with the credentials
func CreateEC2Client(credentials *credentials.Credentials, region string) (*ec2.EC2, error) {

	// set aws log level
	var lv aws.LogLevelType
	if log.Level == logrus.DebugLevel {
		log.Info("set aws log level to debug")
		lv = aws.LogDebug
	} else {
		log.Info("set aws log off")
		lv = aws.LogOff
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials,
		Region:      &region,
		LogLevel:    &lv,
	})
	if err != nil {
		return nil, err
	}

	return ec2.New(sess), nil
}

// CreateAWSCredentials create a 'Credentials' instance from secret's values
func CreateAWSCredentials(values map[string]string) *credentials.Credentials {
	return credentials.NewStaticCredentials(
		values[pkgSecret.AwsAccessKeyId],
		values[pkgSecret.AwsSecretAccessKey],
		"",
	)
}
