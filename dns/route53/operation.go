package route53

type operationType string

const (
	isDomainRegistered operationType = "IsDomainRegistered"
	registerDomain     operationType = "RegisterDomain"
	unregisterDomain   operationType = "UnregisterDomain"
)
