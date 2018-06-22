package route53

import "time"

// status
const (
	CREATING = "CREATING"
	CREATED  = "CREATED"
	FAILED   = "FAILED"
	REMOVING = "REMOVING"
)

// domainState represents the state of a domain registered with Amazon Route53 DNS service
type domainState struct {
	createdAt      time.Time
	organisationId uint
	domain         string
	hostedZoneId   string
	policyArn      string
	iamUser        string
	awsAccessKeyId string
	status         string
	errMsg         string
}
