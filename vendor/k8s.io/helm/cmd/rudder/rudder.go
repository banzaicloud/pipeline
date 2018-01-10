/*
Copyright 2017 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"

	"k8s.io/helm/pkg/kube"
	rudderAPI "k8s.io/helm/pkg/proto/hapi/rudder"
	"k8s.io/helm/pkg/rudder"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/version"
)

var kubeClient *kube.Client
var clientset internalclientset.Interface

func main() {
	var err error
	kubeClient = kube.New(nil)
	clientset, err = kubeClient.ClientSet()
	if err != nil {
		grpclog.Fatalf("Cannot initialize Kubernetes connection: %s", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", rudder.GrpcPort))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	rudderAPI.RegisterReleaseModuleServiceServer(grpcServer, &ReleaseModuleServiceServer{})

	grpclog.Print("Server starting")
	grpcServer.Serve(lis)
	grpclog.Print("Server started")
}

// ReleaseModuleServiceServer provides implementation for rudderAPI.ReleaseModuleServiceServer
type ReleaseModuleServiceServer struct{}

// Version returns Rudder version based on helm version
func (r *ReleaseModuleServiceServer) Version(ctx context.Context, in *rudderAPI.VersionReleaseRequest) (*rudderAPI.VersionReleaseResponse, error) {
	grpclog.Print("version")
	return &rudderAPI.VersionReleaseResponse{
		Name:    "helm-rudder-native",
		Version: version.Version,
	}, nil
}

// InstallRelease creates a release using kubeClient.Create
func (r *ReleaseModuleServiceServer) InstallRelease(ctx context.Context, in *rudderAPI.InstallReleaseRequest) (*rudderAPI.InstallReleaseResponse, error) {
	grpclog.Print("install")
	b := bytes.NewBufferString(in.Release.Manifest)
	err := kubeClient.Create(in.Release.Namespace, b, 500, false)
	if err != nil {
		grpclog.Printf("error when creating release: %v", err)
	}
	return &rudderAPI.InstallReleaseResponse{}, err
}

// DeleteRelease deletes a provided release
func (r *ReleaseModuleServiceServer) DeleteRelease(ctx context.Context, in *rudderAPI.DeleteReleaseRequest) (*rudderAPI.DeleteReleaseResponse, error) {
	grpclog.Print("delete")

	resp := &rudderAPI.DeleteReleaseResponse{}
	rel := in.Release
	vs, err := tiller.GetVersionSet(clientset.Discovery())
	if err != nil {
		return resp, fmt.Errorf("Could not get apiVersions from Kubernetes: %v", err)
	}

	kept, errs := tiller.DeleteRelease(rel, vs, kubeClient)
	rel.Manifest = kept

	allErrors := ""
	for _, e := range errs {
		allErrors = allErrors + "\n" + e.Error()
	}

	if len(allErrors) > 0 {
		err = fmt.Errorf(allErrors)
	}

	return &rudderAPI.DeleteReleaseResponse{
		Release: rel,
	}, err
}

// RollbackRelease rolls back the release
func (r *ReleaseModuleServiceServer) RollbackRelease(ctx context.Context, in *rudderAPI.RollbackReleaseRequest) (*rudderAPI.RollbackReleaseResponse, error) {
	grpclog.Print("rollback")
	c := bytes.NewBufferString(in.Current.Manifest)
	t := bytes.NewBufferString(in.Target.Manifest)
	err := kubeClient.Update(in.Target.Namespace, c, t, in.Force, in.Recreate, in.Timeout, in.Wait)
	return &rudderAPI.RollbackReleaseResponse{}, err
}

// UpgradeRelease upgrades manifests using kubernetes client
func (r *ReleaseModuleServiceServer) UpgradeRelease(ctx context.Context, in *rudderAPI.UpgradeReleaseRequest) (*rudderAPI.UpgradeReleaseResponse, error) {
	grpclog.Print("upgrade")
	c := bytes.NewBufferString(in.Current.Manifest)
	t := bytes.NewBufferString(in.Target.Manifest)
	err := kubeClient.Update(in.Target.Namespace, c, t, in.Force, in.Recreate, in.Timeout, in.Wait)
	// upgrade response object should be changed to include status
	return &rudderAPI.UpgradeReleaseResponse{}, err
}

// ReleaseStatus retrieves release status
func (r *ReleaseModuleServiceServer) ReleaseStatus(ctx context.Context, in *rudderAPI.ReleaseStatusRequest) (*rudderAPI.ReleaseStatusResponse, error) {
	grpclog.Print("status")

	resp, err := kubeClient.Get(in.Release.Namespace, bytes.NewBufferString(in.Release.Manifest))
	in.Release.Info.Status.Resources = resp
	return &rudderAPI.ReleaseStatusResponse{
		Release: in.Release,
		Info:    in.Release.Info,
	}, err
}
