// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package google

import (
	"fmt"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/goph/emperror"
	"golang.org/x/net/context"
	googleauth "golang.org/x/oauth2/google"
	gcp "google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	credentialspb "google.golang.org/genproto/googleapis/iam/credentials/v1"
)

// IamSvc describes the fields needed for interacting with Google's IAM API
type IamSvc struct {
	googleCredentials *googleauth.Credentials
}

// NewIamSvc creates a new IamSvc that will use the specified google credentials for interacting with Google IAM API
func NewIamSvc(googleCredentials *googleauth.Credentials) *IamSvc {
	return &IamSvc{
		googleCredentials: googleCredentials,
	}
}

// GenerateNewAccessToken generates a new GCP access token that expires after the specified duration
func (iam *IamSvc) GenerateNewAccessToken(serviceAccountEmail string, duration *duration.Duration) (*credentialspb.GenerateAccessTokenResponse, error) {
	ctx := context.Background()
	credentialsClient, err := credentials.NewIamCredentialsClient(ctx, option.WithCredentials(iam.googleCredentials))
	if err != nil {
		return nil, emperror.Wrap(err, "instantiating GCP IAM credentials client failed")
	}

	defer credentialsClient.Close()

	// requires Service Account Token Creator and Service Account User IAM roles
	req := credentialspb.GenerateAccessTokenRequest{
		Name:     fmt.Sprintf("projects/-/serviceAccounts/%s", serviceAccountEmail),
		Lifetime: duration,
		Scope: []string{
			gcp.CloudPlatformScope,
			"https://www.googleapis.com/auth/userinfo.email",
		},
	}

	resp, err := credentialsClient.GenerateAccessToken(ctx, &req)
	if err != nil {
		return nil, emperror.WrapWith(err, "generate GCP access token for service account failed", "service account", serviceAccountEmail)
	}
	return resp, nil
}
