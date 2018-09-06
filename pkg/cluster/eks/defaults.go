package eks

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultRegion = UsWest2
)

// DefaultImages in each supported location in EC2 (from https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html)
var DefaultImages = map[string]string{
	UsEast1: "ami-0b2ae3c6bda8b5c06",
	UsWest2: "ami-08cab282f9979fc7a",
}

// EC2 regions
const (
	UsEast1 = "us-east-1"
	UsWest2 = "us-west-2"
)
