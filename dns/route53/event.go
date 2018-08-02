package route53

// DomainEvent holds the common fields for the domain events
type DomainEvent struct {
	Domain         string
	OrganisationId uint
}

// RegisterDomainSucceededEvent is fired when a domain is registered or re-registered in an external DNS service
type RegisterDomainSucceededEvent struct {
	DomainEvent
}

// RegisterDomainFailedEvent is fired when a domain registration or re-registration in an external DNS service
// failed
type RegisterDomainFailedEvent struct {
	DomainEvent
	Cause error
}

// UnregisterDomainSucceededEvent is fired when a domain is un-registered in an external DNS service
type UnregisterDomainSucceededEvent struct {
	DomainEvent
}

// UnregisterDomainFailedEvent is fired when a domain un-registered in an external DNS service
// failed
type UnregisterDomainFailedEvent struct {
	DomainEvent
	Cause error
}
