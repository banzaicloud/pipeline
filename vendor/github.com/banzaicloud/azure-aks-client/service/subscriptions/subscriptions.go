package subscriptions

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest"
)

// Client responsible for listing locations
type Client struct {
	client         *subscriptions.Client
	subscriptionId string
}

// NewClient creates new 'Client' instance
func NewClient(authorizer autorest.Authorizer, subscriptionId string) *Client {
	subscriptionsClient := subscriptions.NewClient()
	subscriptionsClient.Authorizer = authorizer

	return &Client{
		client:         &subscriptionsClient,
		subscriptionId: subscriptionId,
	}
}

// ListLocations provides all the locations that are available for resource providers; however, each
// resource provider may support a subset of this list.
func (c *Client) ListLocations() ([]subscriptions.Location, error) {
	resp, err := c.client.ListLocations(context.Background(), c.subscriptionId)
	if err != nil {
		return nil, err
	}
	return *resp.Value, nil
}

// ListSubscriptions gets all subscriptions for a tenant
func (c *Client) ListSubscriptions() ([]subscriptions.Subscription, error) {
	resp, err := c.client.List(context.Background())
	if err != nil {
		return nil, err
	}

	return resp.Values(), nil
}
