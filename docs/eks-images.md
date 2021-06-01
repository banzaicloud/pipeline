## How to update default EKS images

Default EKS images are listed in [image_selector_defaults.go](https://github.com/banzaicloud/pipeline/blob/master/internal/cluster/distribution/eks/image_selector_defaults.go) for each K8s version and architecture.
You can either [generate this file by running a script](#-Generate) or you can retrieve list of images and update it [manually](#-Manual-update).

### Prerequisites

- Make
- Account on Amazon

You will need an AWS access & secure key with following IAM roles:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeRegions",
                "ssm:Describe*",
                "ssm:Get*",
                "ssm:List*"
            ],
            "Resource": "*"
        }
    ]
}
```

### Generate

1. Check latest available [EKS versions](https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html)
   and update major versions in K8S_VERSIONS list in [generate-eks-image-list.sh](generate-eks-image-list.sh)

1. The script will use your aws keys from `default` profile to retrieve default images.
   You can pass a different profile name as a command parameter:

    ```bash
    ./docs/generate-eks-image-list.sh <YOUR_AWS_PROFILE_NAME>
    ```

1. generate source file containing list of images:

    ```bash
    ./docs/generate-eks-image-list.sh > internal/cluster/distribution/eks/image_selector_defaults.go
    ```

1. run `make fix` to fix eventual source code formatting issues

### Manual update

1. Check latest available [EKS versions](https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html)
   and update major versions in K8S_VERSIONS list in the below script, then run the script.

    ```bash

    K8S_VERSIONS=(
      "1.16"
      "1.17"
      "1.18"
      "1.19"
      "1.20"
    )

    for version in ${K8S_VERSIONS[@]}; do
        echo "K8S Version:" $version
        for region in `aws ec2 describe-regions --output text | cut -f4 | sort -V`; do
            aws ssm get-parameter --name /aws/service/eks/optimized-ami/${version}/amazon-linux-2/recommended/image_id --region ${region} --query Parameter.Value --output text | xargs -I "{}" echo \"$region\": \"{}\",
        done
    done

    for version in ${K8S_VERSIONS[@]}; do
        echo "K8S Version (GPU accelerated):" $version
        for region in `aws ec2 describe-regions --output text | cut -f4 | sort -V`; do
            aws ssm get-parameter --name /aws/service/eks/optimized-ami/${version}/amazon-linux-2-gpu/recommended/image_id --region ${region} --query Parameter.Value --output text | xargs -I "{}" echo \"$region\": \"{}\",
        done
    done

    for version in ${K8S_VERSIONS[@]}; do
        echo "K8S Version (ARM):" $version
        for region in `aws ec2 describe-regions --output text | cut -f4 | sort -V`; do
            aws ssm get-parameter --name /aws/service/eks/optimized-ami/${version}/amazon-linux-2-arm64/recommended/image_id --region ${region} --query Parameter.Value --output text | xargs -I "{}" echo \"$region\": \"{}\",
        done
    done
    ```

1. Make sure to update `defaultImages`, `defaultAcceleratedImages`, `defaultARMImages` in [image_selector_defaults.go](https://github.com/banzaicloud/pipeline/blob/master/internal/cluster/distribution/eks/image_selector_defaults.go)
