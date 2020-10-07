// Copyright © 2020 Banzai Cloud
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

package types

import (
	"github.com/banzaicloud/pipeline/internal/secret"
)

const Vsphere = "vsphere"

const (
	FieldVsphereURL                 = "url"
	FieldVsphereUser                = "user"
	FieldVspherePassword            = "password"
	FieldVsphereFingerprint         = "fingerprint"
	FieldVsphereDatacenter          = "datacenter"
	FieldVsphereDatastore           = "datastore"
	FieldVsphereResourcePool        = "resourcePool"
	FieldVsphereFolder              = "folder"
	FieldVsphereDefaultNodeTemplate = "defaultNodeTemplate"
)

type VsphereType struct{}

func (VsphereType) Name() string {
	return Vsphere
}

func (VsphereType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldVsphereURL, Required: true, IsSafeToDisplay: true, Description: "The URL endpoint of the vSphere instance to use (don't include auth info)"},
			{Name: FieldVsphereUser, Required: true, IsSafeToDisplay: true, Description: "Username to use for vSphere authentication"},
			{Name: FieldVspherePassword, Required: true, Description: "Password to use for vSphere authentication"},
			{Name: FieldVsphereFingerprint, Required: true, IsSafeToDisplay: true, Description: "Fingerprint of the server certificate of vCenter"},
			{Name: FieldVsphereDatacenter, Required: true, IsSafeToDisplay: true, Description: "Datacenter to use to store persistent volumes"},
			{Name: FieldVsphereDatastore, Required: true, IsSafeToDisplay: true, Description: "Datastore that is in the given datacenter, and is available on all nodes"},
			{Name: FieldVsphereResourcePool, Required: true, IsSafeToDisplay: true, Description: "Resource pool to create  VMs"},
			{Name: FieldVsphereFolder, Required: true, IsSafeToDisplay: true, Description: "The name of the folder (aka blue folder) to create VMs"},
			{Name: FieldVsphereDefaultNodeTemplate, Required: true, IsSafeToDisplay: true, Description: "The name of the default template name for VMs"},
		},
	}
}

func (t VsphereType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}
