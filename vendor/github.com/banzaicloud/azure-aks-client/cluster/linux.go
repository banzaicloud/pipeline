package cluster

type LinuxProfile struct {
	AdminUsername string `json:"adminUsername"`
	SSH           SSH    `json:"ssh"`
}

type SSHPublicKeys struct {
	KeyData *string `json:"keyData,omitempty"`
}

type SSH struct {
	PublicKeys *[]SSHPublicKey `json:"publicKeys,omitempty"`
}
