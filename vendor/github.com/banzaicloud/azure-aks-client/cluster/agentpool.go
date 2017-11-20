package cluster

type AgentPoolProfiles struct {
	Count          int    `json:"count"`
	DNSPrefix      string `json:"dnsPrefix"`
	Fqdn           string `json:"fqdn"`
	Name           string `json:"name"`
	OsType         string `json:"osType"`
	Ports          []int  `json:"ports"`
	StorageProfile string `json:"storageProfile"`
	VMSize         string `json:"vmSize"`
}
