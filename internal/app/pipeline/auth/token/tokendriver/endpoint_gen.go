// Code generated by mga tool. DO NOT EDIT.
package tokendriver

import (
	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
	"github.com/go-kit/kit/endpoint"
	kitoc "github.com/go-kit/kit/tracing/opencensus"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
)

// Endpoints collects all of the endpoints that compose the underlying service. It's
// meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	CreateToken endpoint.Endpoint
	DeleteToken endpoint.Endpoint
	GetToken    endpoint.Endpoint
	ListTokens  endpoint.Endpoint
}

// MakeEndpoints returns a(n) Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeEndpoints(service token.Service, middleware ...endpoint.Middleware) Endpoints {
	mw := kitxendpoint.Chain(middleware...)

	return Endpoints{
		CreateToken: mw(MakeCreateTokenEndpoint(service)),
		DeleteToken: mw(MakeDeleteTokenEndpoint(service)),
		GetToken:    mw(MakeGetTokenEndpoint(service)),
		ListTokens:  mw(MakeListTokensEndpoint(service)),
	}
}

// TraceEndpoints returns a(n) Endpoints struct where each endpoint is wrapped with a tracing middleware.
func TraceEndpoints(endpoints Endpoints) Endpoints {
	return Endpoints{
		CreateToken: kitoc.TraceEndpoint("token.CreateToken")(endpoints.CreateToken),
		DeleteToken: kitoc.TraceEndpoint("token.DeleteToken")(endpoints.DeleteToken),
		GetToken:    kitoc.TraceEndpoint("token.GetToken")(endpoints.GetToken),
		ListTokens:  kitoc.TraceEndpoint("token.ListTokens")(endpoints.ListTokens),
	}
}
