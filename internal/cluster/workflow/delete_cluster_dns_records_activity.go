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

package workflow

import (
	"context"

	"github.com/goph/emperror"
)

const DeleteClusterDNSRecordsActivityName = "delete-cluster-dns-records"

type DeleteClusterDNSRecordsActivityInput struct {
	OrganizationID uint
	ClusterUID     string
}

type DeleteClusterDNSRecordsActivity struct {
	deleter ClusterDNSRecordsDeleter
}

type ClusterDNSRecordsDeleter interface {
	Delete(organizationID uint, clusterUID string) error
}

func MakeDeleteClusterDNSRecordsActivity(deleter ClusterDNSRecordsDeleter) DeleteClusterDNSRecordsActivity {
	return DeleteClusterDNSRecordsActivity{
		deleter: deleter,
	}
}

func (a DeleteClusterDNSRecordsActivity) Execute(ctx context.Context, input DeleteClusterDNSRecordsActivityInput) error {
	return emperror.Wrap(a.deleter.Delete(input.OrganizationID, input.ClusterUID), "failed to delete cluster DNS records")
}
