package route53

// awsRoute53StateStore manages the state of the domains
// registered by us in the Amazon Route53 external DNS service
type awsRoute53StateStore interface {
	create(state *domainState) error
	update(state *domainState) error
	find(orgId uint, domain string, state *domainState) (bool, error)
	listUnused() ([]domainState, error)
	delete(state *domainState) error
}
