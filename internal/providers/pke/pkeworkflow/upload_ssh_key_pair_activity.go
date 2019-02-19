package pkeworkflow

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const UploadSSHKeyPairActivityName = "pke-upload-ssh-key-pair-activity"

type UploadSSHKeyPairActivity struct {
	clusters Clusters
}

func NewUploadSSHKeyPairActivity(clusters Clusters) *UploadSSHKeyPairActivity {
	return &UploadSSHKeyPairActivity{
		clusters: clusters,
	}
}

type UploadSSHKeyPairActivityInput struct {
	ClusterID uint
}

type UploadSSHKeyPairActivityOutput struct {
	KeyName string
}

func (a *UploadSSHKeyPairActivity) Execute(ctx context.Context, input UploadSSHKeyPairActivityInput) (*UploadSSHKeyPairActivityOutput, error) {
	//log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return nil, err
	}
	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return nil, errors.New(fmt.Sprintf("can't create VPC for cluster type %t", c))
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to connect to AWS")
	}

	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return nil, err
	}

	clusterName := cluster.GetName()
	keyName := "pke-ssh-" + clusterName

	e := ec2.New(client)

	describeKeyPairsInput := &ec2.DescribeKeyPairsInput{
		KeyNames: aws.StringSlice([]string{keyName}),
	}

	describeKeyPairsOutput, err := e.DescribeKeyPairs(describeKeyPairsInput)
	if err != nil {
		if a, ok := err.(awserr.Error); ok {
			if a.Code() != "InvalidKeyPair.NotFound" {
				return nil, a
			}
		} else {
			return nil, err
		}
	}

	if len(describeKeyPairsOutput.KeyPairs) > 0 {
		// key already exists
		return &UploadSSHKeyPairActivityOutput{
			KeyName: keyName,
		}, nil
	}

	publicKey, err := cluster.GetSshPublicKey()
	if err != nil {
		return nil, err
	}

	importKeyPairInput := &ec2.ImportKeyPairInput{
		KeyName:           &keyName,
		PublicKeyMaterial: []byte(publicKey),
	}
	importKeyPairOutput, err := e.ImportKeyPair(importKeyPairInput)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to import key pair on AWS EC2")
	}

	return &UploadSSHKeyPairActivityOutput{
		KeyName: *importKeyPairOutput.KeyName,
	}, nil
}
