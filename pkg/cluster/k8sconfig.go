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

package cluster

import (
	"fmt"
	"strings"

	"emperror.dev/emperror"
	"gopkg.in/yaml.v2"
)

type KubeConfig struct {
	APIVersion     string          `yaml:"apiVersion,omitempty"`
	Clusters       []configCluster `yaml:"clusters,omitempty"`
	Contexts       []configContext `yaml:"contexts,omitempty"`
	Users          []configUser    `yaml:"users,omitempty"`
	CurrentContext string          `yaml:"current-context,omitempty"`
	Kind           string          `yaml:"kind,omitempty"`
	// Kubernetes config contains an invalid map for the go yaml parser,
	// preferences field always look like this {} this should be {{}} so
	// yaml.Unmarshal fails with a cryptic error message which says string
	// cannot be casted as !map
	// Preferences    string          `yaml:"preferences,omitempty"`
}

type configCluster struct {
	Cluster dataCluster `yaml:"cluster,omitempty"`
	Name    string      `yaml:"name,omitempty"`
}

type configContext struct {
	Context contextData `yaml:"context,omitempty"`
	Name    string      `yaml:"name,omitempty"`
}

type contextData struct {
	Cluster string `yaml:"cluster,omitempty"`
	User    string `yaml:"user,omitempty"`
}

type configUser struct {
	Name string   `yaml:"name,omitempty"`
	User userData `yaml:"user,omitempty"`
}

type userData struct {
	Token                 string       `yaml:"token,omitempty"`
	Username              string       `yaml:"username,omitempty"`
	Password              string       `yaml:"password,omitempty"`
	ClientCertificateData string       `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string       `yaml:"client-key-data,omitempty"`
	AuthProvider          authProvider `yaml:"auth-provider,omitempty"`
}

type authProvider struct {
	ProviderConfig providerConfig `yaml:"config,omitempty"`
	Name           string         `yaml:"name,omitempty"`
}

type providerConfig struct {
	AccessToken string `yaml:"access-token,omitempty"`
	Expiry      string `yaml:"expiry,omitempty"`
}

type dataCluster struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	Server                   string `yaml:"server,omitempty"`
}

// StoreConfig returns marshalled config file
func StoreConfig(c *KubernetesCluster) ([]byte, error) {
	isBasicOn := false
	if c.Username != "" && c.Password != "" {
		isBasicOn = true
	}
	username, password, token := "", "", ""
	if isBasicOn {
		username = c.Username
		password = c.Password
	} else {
		token = c.ServiceAccountToken
	}

	config := KubeConfig{}
	config.APIVersion = "v1"
	config.Kind = "Config"

	// setup clusters
	host := c.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}
	cluster := configCluster{
		Cluster: dataCluster{
			CertificateAuthorityData: string(c.RootCACert),
			Server:                   host,
		},
		Name: c.Name,
	}
	if config.Clusters == nil || len(config.Clusters) == 0 {
		config.Clusters = []configCluster{cluster}
	} else {
		exist := false
		for _, cluster := range config.Clusters {
			if cluster.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Clusters = append(config.Clusters, cluster)
		}
	}

	var provider authProvider
	if len(c.AuthProviderName) != 0 || len(c.AuthAccessToken) != 0 {
		provider = authProvider{
			ProviderConfig: providerConfig{
				AccessToken: c.AuthAccessToken,
				Expiry:      c.AuthAccessTokenExpiry,
			},
			Name: c.AuthProviderName,
		}
	}

	// setup users
	user := configUser{
		User: userData{
			Token:        token,
			Username:     username,
			Password:     password,
			AuthProvider: provider,
		},
		Name: c.Name,
	}
	if config.Users == nil || len(config.Users) == 0 {
		config.Users = []configUser{user}
	} else {
		exist := false
		for _, user := range config.Users {
			if user.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Users = append(config.Users, user)
		}
	}

	// setup context
	context := configContext{
		Context: contextData{
			Cluster: c.Name,
			User:    c.Name,
		},
		Name: c.Name,
	}

	config.CurrentContext = context.Name

	if config.Contexts == nil || len(config.Contexts) == 0 {
		config.Contexts = []configContext{context}
	} else {
		exist := false
		for _, context := range config.Contexts {
			if context.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Contexts = append(config.Contexts, context)
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, emperror.Wrap(err, "marshaling kubernetes config failed")
	}

	return data, nil
}

// KubernetesCluster represents a kubernetes cluster
type KubernetesCluster struct {
	// The name of the cluster
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// The status of the cluster
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	// Kubernetes cluster version
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Service account token to access kubernetes API
	ServiceAccountToken string `json:"serviceAccountToken,omitempty" yaml:"service_account_token,omitempty"`
	// Kubernetes API master endpoint
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	// Username for http basic authentication
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	// Password for http basic authentication
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// Root CaCertificate for API server(base64 encoded)
	RootCACert string `json:"rootCACert,omitempty" yaml:"root_ca_cert,omitempty"`
	// Client Certificate(base64 encoded)
	ClientCertificate string `json:"clientCertificate,omitempty" yaml:"client_certificate,omitempty"`
	// Client private key(base64 encoded)
	ClientKey string `json:"clientKey,omitempty" yaml:"client_key,omitempty"`
	// Node count in the cluster
	NodeCount int64 `json:"nodeCount,omitempty" yaml:"node_count,omitempty"`
	// Metadata store specific driver options per cloud provider
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	AuthProviderName      string `json:"auth_provider_name,omitempty"`
	AuthAccessToken       string `json:"auth_access_token,omitempty"`
	AuthAccessTokenExpiry string `json:"auth_access_token_expiry,omitempty"`
	CurrentContext        string `json:"current_context,omitempty"`
}

func GetAPIEndpointFromKubeconfig(config []byte) (string, error) {
	kubeConf := KubeConfig{}
	err := yaml.Unmarshal(config, &kubeConf)
	if err != nil {
		return "", emperror.Wrap(err, "failed to parse cluster's Kubeconfig")
	}

	return kubeConf.Clusters[0].Cluster.Server, nil
}

// CreateDummyConfig creates a dummy kubeconfig
func CreateDummyConfig() *KubeConfig {
	return &KubeConfig{
		APIVersion: "v1",
		Clusters: []configCluster{
			{
				Cluster: dataCluster{
					Server: "http://cow.org:8080",
				},
				Name: "cow-cluster",
			}, {
				Cluster: dataCluster{
					Server: "https://horse.org:4443",
				},
				Name: "horse-cluster",
			},
			{
				Cluster: dataCluster{
					Server: "https://pig.org:443",
				},
				Name: "pig-cluster",
			},
		},
		Contexts: []configContext{
			{
				Context: contextData{
					Cluster: "horse-cluster",
					User:    "green-user",
				},
				Name: "federal-context",
			}, {
				Context: contextData{
					Cluster: "pig-cluster",
					User:    "black-user",
				},
				Name: "queen-anne-context",
			},
		},
		Users: []configUser{
			{
				Name: "blue-user",
				User: userData{
					Token: "blue-token",
				},
			},
			{
				Name: "green-user",
				User: userData{},
			},
		},
		CurrentContext: "federal-context",
		Kind:           "Config",
	}

}
