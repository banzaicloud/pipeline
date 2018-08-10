package eks

// ### [ Constants to EKS cluster default values ] ### //
const (
	DefaultRegion = UsWest2
)

// DefaultImages in each supported location in EC2 (from https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html)
var DefaultImages = map[string]string{
	UsEast1: "ami-0fef2bff3c2e2da93",
	UsWest2: "ami-0ea01e1d1dea65b5c",
}

// EC2 regions
const (
	UsEast1 = "us-east-1"
	UsWest2 = "us-west-2"
)
