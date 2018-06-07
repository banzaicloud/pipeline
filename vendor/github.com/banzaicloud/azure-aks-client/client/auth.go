package client

import (
	"github.com/Azure/go-autorest/autorest/adal"
	"fmt"
)

const (
	// Active Directory Endpoint
	activeDirectoryEndpoint = "https://login.microsoftonline.com/"

	// The resource for which the token is acquired
	resource = "https://management.core.windows.net/"
)

// validateCredentials validates all credentials
func (a *aksClient) validateCredentials() (err error) {

	subscriptions, err := a.listSubscriptions()
	if err != nil {
		return
	}

	found := false
	for _, s := range subscriptions {
		if *s.SubscriptionID == a.azureSdk.ServicePrincipal.SubscriptionID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("the subscription '%s' could not be found", a.azureSdk.ServicePrincipal.SubscriptionID)
	}

	return a.refreshToken()
}

// refreshToken obtains a fresh token for the Service Principal.
func (a *aksClient) refreshToken() (err error) {

	tenantID := a.azureSdk.ServicePrincipal.TenantId
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)

	token, err := adal.NewServicePrincipalToken(*oauthConfig, a.azureSdk.ServicePrincipal.ClientID, a.azureSdk.ServicePrincipal.ClientSecret, resource)
	if err != nil {
		return
	}

	return token.Refresh()
}
