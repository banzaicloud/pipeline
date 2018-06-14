package route53

// status
const (
	CREATING = "CREATING"
	CREATED  = "CREATED"
	FAILED   = "FAILED"
	REMOVING = "REMOVING"
)

// domainState represents the state of a domain registered with Amazon Route53 DNS service
type domainState struct {
	organisationId uint
	domain         string
	hostedZoneId   string
	policyArn      string
	iamUser        string
	awsAccessKeyId string
	status         string
	errMsg         string
}
