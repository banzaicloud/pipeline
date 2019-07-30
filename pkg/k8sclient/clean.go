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

package k8sclient

import (
	"emperror.dev/errors"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// cleanConfig collects the minimum needed and supported info from a Kubeconfig structure
func cleanConfig(in *clientcmdapi.Config) (*clientcmdapi.Config, error) {
	name := in.CurrentContext

	context, ok := in.Contexts[name]
	if !ok || context == nil {
		return nil, errors.New("can't find referenced current context in Kubeconfig")
	}

	context, err := cleanContext(context)
	if err != nil {
		return nil, errors.WrapIf(err, "invalid context block in Kubeconfig")
	}

	authinfo, ok := in.AuthInfos[context.AuthInfo]
	if !ok || authinfo == nil {
		return nil, errors.New("can't find referenced user in Kubeconfig")
	}

	authinfo, err = cleanAuthInfo(authinfo)
	if err != nil {
		return nil, errors.WrapIf(err, "invalid user block in Kubeconfig")
	}

	cluster, ok := in.Clusters[context.Cluster]
	if !ok || cluster == nil {
		return nil, errors.New("can't find referenced cluster in Kubeconfig")
	}

	cluster, err = cleanCluster(cluster)
	if err != nil {
		return nil, errors.WrapIf(err, "invalid cluster block in Kubeconfig")
	}

	out := clientcmdapi.Config{
		// Kind -- not needed
		// APIVersion -- not needed
		// Preferences -- not needed
		CurrentContext: name,
		Contexts:       map[string]*clientcmdapi.Context{name: context},
		Clusters:       map[string]*clientcmdapi.Cluster{context.Cluster: cluster},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{context.AuthInfo: authinfo},
		// Extensions -- not needed
	}
	return &out, nil
}

func cleanContext(in *clientcmdapi.Context) (*clientcmdapi.Context, error) {
	out := clientcmdapi.Context{
		// LocationOfOrigin -- not needed
		Cluster:   in.Cluster,
		AuthInfo:  in.AuthInfo,
		Namespace: in.Namespace,
		// Extensions -- not needed
	}
	return &out, nil
}

func cleanAuthInfo(in *clientcmdapi.AuthInfo) (*clientcmdapi.AuthInfo, error) {
	execConfig, err := cleanExecConfig(in.Exec)
	if err != nil {
		return nil, errors.WrapIf(err, "invalid exec field")
	}

	switch {
	case in.ClientCertificate != "":
		return nil, errors.New("client certificate files are not supported")
	case in.ClientKey != "":
		return nil, errors.New("client key files are not supported")
	case in.TokenFile != "":
		return nil, errors.New("token files are not supported")
	case in.AuthProvider != nil && len(in.AuthProvider.Config) > 0:
		return nil, errors.New("auth provider configurations are not supported (try exec instead)")
	case in.Impersonate != "" || len(in.ImpersonateGroups) > 0:
		return nil, errors.New("impoersonation is not supported")
	}

	out := clientcmdapi.AuthInfo{
		// LocationOfOrigin -- not needed
		// ClientCertificate -- reads fs
		ClientCertificateData: in.ClientCertificateData,
		// ClientKey -- reads fs
		ClientKeyData: in.ClientKeyData,
		Token:         in.Token,
		// TokenFile -- reads fs
		// Impersonate -- not needed
		// ImpersonateGroups -- not needed
		// ImpersonateUserExtra -- not needed
		Username: in.Username,
		Password: in.Password,
		// AuthProvider -- potentionally insecure, not supported
		Exec: execConfig,
		// Extensions -- not needed
	}
	return &out, nil
}

func cleanExecConfig(in *clientcmdapi.ExecConfig) (*clientcmdapi.ExecConfig, error) {
	if in == nil {
		return nil, nil
	}
	if in.Command != "aws-iam-authenticator" {
		return nil, errors.Errorf("unsupported authenticator command: %q", in.Command)
	}

	out := clientcmdapi.ExecConfig{
		Command:    in.Command,
		Args:       in.Args,
		Env:        in.Env,
		APIVersion: in.APIVersion,
	}
	return &out, nil
}

func cleanCluster(in *clientcmdapi.Cluster) (*clientcmdapi.Cluster, error) {
	switch {
	case in.CertificateAuthority != "":
		return nil, errors.New("CA files are not supported")
	}

	out := clientcmdapi.Cluster{
		// LocationOfOrigin -- not needed
		Server:                in.Server, // TODO: find out if it's not localhost, cloud api, etc in transport layer (there would be a race condition in dns resolution if checked here)
		InsecureSkipTLSVerify: in.InsecureSkipTLSVerify,
		// CertificateAuthority -- reads fs
		CertificateAuthorityData: in.CertificateAuthorityData,
		// Extensions -- not needed
	}
	return &out, nil
}

// CleanKubeconfig cleans up a serialized kubeconfig and returns it in the same format
func CleanKubeconfig(in []byte) ([]byte, error) {
	apiconfig, err := clientcmd.Load(in)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to load kubernetes API config")
	}

	apiconfig, err = cleanConfig(apiconfig)
	if err != nil {
		return nil, err
	}

	out, err := clientcmd.Write(*apiconfig)
	return out, errors.WrapIf(err, "failed to write cleaned kubernetes API config")
}
