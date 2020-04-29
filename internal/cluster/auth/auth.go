// Copyright © 2019 Banzai Cloud
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

package auth

import (
	"context"
	"encoding/base64"
	"net/url"

	"emperror.dev/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	k8sClient "k8s.io/client-go/tools/clientcmd"
	k8sClientApi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/banzaicloud/pipeline/.gen/dex"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
)

const authSecretName = "dex-client"
const configSecretName = "config"
const clientIDKey = "clientID"
const clientSecretKey = "clientSecret"

type ClusterClientSecret struct {
	ClientID     string
	ClientSecret string
}

type dexClient struct {
	dex.DexClient
	grpcConn *grpc.ClientConn
}

func (d *dexClient) Close() error {
	return d.grpcConn.Close()
}

func newDexClient(hostAndPort, caPath string) (*dexClient, error) {
	dialOption := grpc.WithInsecure()
	if caPath != "" {
		creds, err := credentials.NewClientTLSFromFile(caPath, "")
		if err != nil {
			return nil, errors.WrapIff(err, "loading dex CA cert failed")
		}
		dialOption = grpc.WithTransportCredentials(creds)
	}
	conn, err := grpc.Dial(hostAndPort, dialOption)
	if err != nil {
		return nil, errors.WrapIff(err, "grpc dial failed")
	}
	return &dexClient{DexClient: dex.NewDexClient(conn), grpcConn: conn}, nil
}

type ClusterAuthService interface {
	RegisterCluster(context.Context, string, uint, string) error
	UnRegisterCluster(context.Context, string) error
	GetClusterConfig(context.Context, uint) (*k8sClientApi.Config, error)

	ClusterClientSecretGetter
}

type ClusterClientSecretGetter interface {
	GetClusterClientSecret(context.Context, uint) (ClusterClientSecret, error)
}

type noOpClusterAuthService struct {
}

func NewNoOpClusterAuthService() (ClusterAuthService, error) {
	return &noOpClusterAuthService{}, nil
}

func (*noOpClusterAuthService) RegisterCluster(tx context.Context, clusterName string, clusterID uint, clusterUID string) error {
	return nil
}

func (*noOpClusterAuthService) UnRegisterCluster(tx context.Context, clusterUID string) error {
	return nil
}

func (*noOpClusterAuthService) GetClusterClientSecret(ctx context.Context, clusterID uint) (ClusterClientSecret, error) {
	return ClusterClientSecret{}, nil
}

func (*noOpClusterAuthService) GetClusterConfig(ctx context.Context, clusterID uint) (*k8sClientApi.Config, error) {
	return nil, nil
}

type dexClusterAuthService struct {
	dexClient           *dexClient
	secretStore         *clustersecret.Store
	pipelineRedirectURI string
}

func NewDexClusterAuthService(secretStore *clustersecret.Store) (ClusterAuthService, error) {
	client, err := newDexClient(global.Config.Dex.APIAddr, global.Config.Dex.APICa)
	if err != nil {
		return nil, errors.WrapIff(err, "failed to create dex auth service")
	}

	pipelineExternalURL, err := url.Parse(global.Config.Pipeline.External.URL)
	if err != nil {
		return nil, errors.WrapIff(err, "failed to parse pipeline externalURL")
	}

	pipelineExternalURL.Path = "/auth/dex/cluster/callback"

	return &dexClusterAuthService{
		dexClient:           client,
		secretStore:         secretStore,
		pipelineRedirectURI: pipelineExternalURL.String(),
	}, nil
}

func (a *dexClusterAuthService) RegisterCluster(ctx context.Context, clusterName string, clusterID uint, clusterUID string) error {
	clientID := clusterUID
	clientSecret, _ := secret.RandomString("randAlphaNum", 32)
	cliRedirectURI := "http://localhost:5555/callback"
	pipelineRedirectURI := a.pipelineRedirectURI

	req := &dex.CreateClientReq{
		Client: &dex.Client{
			Id:     clientID,
			Name:   clusterName,
			Secret: clientSecret,
			RedirectUris: []string{
				cliRedirectURI,
				pipelineRedirectURI,
			},
		},
	}

	if _, err := a.dexClient.CreateClient(ctx, req); err != nil {
		return errors.WrapIff(err, "failed to create dex client for cluster: %s", clusterUID)
	}

	// save the secret to the secret store
	secretRequest := clustersecret.SecretCreateRequest{
		Type: secrettype.GenericSecret,
		Name: authSecretName,
		Values: map[string]string{
			clientIDKey:     clientID,
			clientSecretKey: clientSecret,
		},
	}

	_, err := a.secretStore.EnsureSecretExists(ctx, clusterID, secretRequest)

	if err != nil {
		return errors.WrapIff(err, "failed to create secret for dex clientID/clientSecret for cluster: %s", clusterUID)
	}

	return nil
}

func (a *dexClusterAuthService) UnRegisterCluster(ctx context.Context, clusterUID string) error {
	clientID := clusterUID

	req := &dex.DeleteClientReq{
		Id: clientID,
	}

	if _, err := a.dexClient.DeleteClient(ctx, req); err != nil {
		return errors.WrapIff(err, "failed to delete dex client for cluster: %s", clusterUID)
	}

	return nil
}

func (a *dexClusterAuthService) GetClusterClientSecret(ctx context.Context, clusterID uint) (ClusterClientSecret, error) {
	secret, err := a.secretStore.GetSecret(ctx, clusterID, authSecretName)

	if err != nil {
		return ClusterClientSecret{}, errors.WrapIff(err, "failed to get dex client for cluster: %d", clusterID)
	}

	return ClusterClientSecret{
		ClientID:     secret.Values[clientIDKey],
		ClientSecret: secret.Values[clientSecretKey],
	}, nil
}

func (a *dexClusterAuthService) GetClusterConfig(ctx context.Context, clusterID uint) (*k8sClientApi.Config, error) {
	secret, err := a.secretStore.GetSecret(ctx, clusterID, configSecretName)
	if err != nil {
		return nil, errors.WrapIff(err, "failed to get dex client for cluster: %d", clusterID)
	}

	configData, err := base64.StdEncoding.DecodeString(secret.Values[secrettype.K8SConfig])
	if err != nil {
		return nil, errors.WrapIff(err, "failed to base64 decode kubeconfig: %d", clusterID)
	}

	return k8sClient.Load(configData)
}
