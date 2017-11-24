package cluster

type Properties struct {
	DNSPrefix         string `json:"dnsPrefix"`
	Fqdn              string `json:"fqdn"`
	KubernetesVersion string `json:"kubernetesVersion"`
	//ProvisioningState   	string `json:"provisioningState"`

	AccessProfiles          AccessProfiles          `json:"accessProfiles"`
	AgentPoolProfiles       []AgentPoolProfiles     `json:"agentPoolProfiles"`
	LinuxProfile            LinuxProfile            `json:"linuxProfile"`
	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile"`
}
