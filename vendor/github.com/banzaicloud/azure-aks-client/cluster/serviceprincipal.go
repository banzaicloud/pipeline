package cluster

type ServicePrincipalProfile struct {
	ClientID *string `json:"clientId,omitempty"`
	Secret   *string `json:"secret,omitempty"`
}

// SSHConfiguration is SSH configuration for Linux-based VMs running on Azure.
type SSHConfiguration struct {
	PublicKeys *[]SSHPublicKey `json:"publicKeys,omitempty"`
}

// SSHPublicKey is contains information about SSH certificate public key data.
type SSHPublicKey struct {
	KeyData *string `json:"keyData,omitempty"`
}
