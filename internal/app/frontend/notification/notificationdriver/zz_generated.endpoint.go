// +build !ignore_autogenerated

// Code generated by mga tool. DO NOT EDIT.

package notificationdriver

import (
	"context"
	"errors"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
	"github.com/go-kit/kit/endpoint"
	kitoc "github.com/go-kit/kit/tracing/opencensus"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
)

// endpointError identifies an error that should be returned as an endpoint error.
type endpointError interface {
	EndpointError() bool
}

// serviceError identifies an error that should be returned as a service error.
type serviceError interface {
	ServiceError() bool
}

// Endpoints collects all of the endpoints that compose the underlying service. It's
// meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	GetNotifications endpoint.Endpoint
}

// MakeEndpoints returns a(n) Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeEndpoints(service notification.Service, middleware ...endpoint.Middleware) Endpoints {
	mw := kitxendpoint.Combine(middleware...)

	return Endpoints{GetNotifications: kitxendpoint.OperationNameMiddleware("notification.GetNotifications")(mw(MakeGetNotificationsEndpoint(service)))}
}

// TraceEndpoints returns a(n) Endpoints struct where each endpoint is wrapped with a tracing middleware.
func TraceEndpoints(endpoints Endpoints) Endpoints {
	return Endpoints{GetNotifications: kitoc.TraceEndpoint("notification.GetNotifications")(endpoints.GetNotifications)}
}

// GetNotificationsRequest is a request struct for GetNotifications endpoint.
type GetNotificationsRequest struct{}

// GetNotificationsResponse is a response struct for GetNotifications endpoint.
type GetNotificationsResponse struct {
	Notifications notification.Notifications
	Err           error
}

func (r GetNotificationsResponse) Failed() error {
	return r.Err
}

// MakeGetNotificationsEndpoint returns an endpoint for the matching method of the underlying service.
func MakeGetNotificationsEndpoint(service notification.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		notifications, err := service.GetNotifications(ctx)

		if err != nil {
			if endpointErr := endpointError(nil); errors.As(err, &endpointErr) && endpointErr.EndpointError() {
				return GetNotificationsResponse{
					Err:           err,
					Notifications: notifications,
				}, err
			}

			return GetNotificationsResponse{
				Err:           err,
				Notifications: notifications,
			}, nil
		}

		return GetNotificationsResponse{Notifications: notifications}, nil
	}
}
