package pke

type NetworkCloudProviderConfigAmazon struct {
	VPCID   string  `yaml:"vpcID"`
	Subnets Subnets `yaml:"subnets"`
}
