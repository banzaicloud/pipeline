#bash

K8S_VERSIONS=(
  "1.16"
  "1.17"
  "1.18"
  "1.19"
  "1.20"
)

AWS_REGIONS=$(aws ec2 describe-regions --output text | cut -f4 | sort -V)

AWS_PROFILE_NAME="${1:-default}"

function print_images () {

  echo "// nolint: gochecknoglobals"
  echo "var $1 = ImageSelectors{"

  for version in ${K8S_VERSIONS[@]}; do
    echo "\\tKubernetesVersionImageSelector{ // Kubernetes Version $version"
    echo "\\t\\tConstraint: mustConstraint(\"$version\"),"
    echo "\\t\\tImageSelector: RegionMapImageSelector{"
  	for region in ${AWS_REGIONS[@]}; do
  	    AWS_PROFILE=$AWS_PROFILE_NAME aws ssm get-parameter --name /aws/service/eks/optimized-ami/${version}/$2/recommended/image_id --region ${region} --query Parameter.Value --output text | xargs -I "{}" printf "\\t\\t\\t%s : %s,\\n" \"$region\" \"{}\"
  	done
    echo "\\t\\t},"
    echo "\\t},"
  done

  echo "}"
}

echo "// Copyright © 2020 Banzai Cloud"
echo "//"
echo "// Licensed under the Apache License, Version 2.0 (the \"License\");"
echo "// you may not use this file except in compliance with the License."
echo "// You may obtain a copy of the License at"
echo "//"
echo "//     http://www.apache.org/licenses/LICENSE-2.0"
echo "//"
echo "// Unless required by applicable law or agreed to in writing, software"
echo "// distributed under the License is distributed on an \"AS IS\" BASIS,"
echo "// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied."
echo "// See the License for the specific language governing permissions and"
echo "// limitations under the License."
echo ""
echo "package eks"
echo ""



echo "// AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html"
print_images "defaultImages" "amazon-linux-2"

echo ""
echo "// GPU accelerated AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html"
print_images "defaultAcceleratedImages" "amazon-linux-2-gpu"

echo ""
echo "// ARM architecture AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html"
print_images "defaultARMImages" "amazon-linux-2-arm64"
