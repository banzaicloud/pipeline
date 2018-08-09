package ec2

// ### [ Constants to Amazon cluster default values ] ### //
const (
	DefaultInstanceType = "m4.xlarge"
	DefaultSpotPrice    = "0.2"
	DefaultRegion       = EuWest1
)

// DefaultImages in each supported location in EC2
var DefaultImages = map[string]string{
	ApNortheast1: "ami-84f19869",
	ApNortheast2: "ami-00be096e",
	ApSouth1:     "ami-1162517e",
	ApSouthEast1: "ami-d385c039",
	ApSouthEast2: "ami-4719bd25",
	CaCentral1:   "ami-5a88053e",
	EuCentral1:   "ami-2bfcfdc0",
	EuWest1:      "ami-4d485ca7",
	EuWest2:      "ami-72709b15",
	EuWest3:      "ami-619a2a1c",
	SaEast1:      "ami-64a78108",
	UsWest1:      "ami-1c7b997f",
	UsWest2:      "ami-e7d28e9f",
}

// EC2 regions
const (
	ApNortheast1 = "ap-northeast-1"
	ApNortheast2 = "ap-northeast-2"
	ApSouth1     = "ap-south-1"
	ApSouthEast1 = "ap-southeast-1"
	ApSouthEast2 = "ap-southeast-2"
	CaCentral1   = "ca-central-1"
	EuCentral1   = "eu-central-1"
	EuWest1      = "eu-west-1"
	EuWest2      = "eu-west-2"
	EuWest3      = "eu-west-3"
	SaEast1      = "sa-east-1"
	UsWest1      = "us-west-1"
	UsWest2      = "us-west-2"
)
