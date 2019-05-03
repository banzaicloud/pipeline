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

package dns

import (
	"github.com/banzaicloud/pipeline/dns"
	"github.com/goph/emperror"
)

type RecordsDeleter struct {
	dnsSVC dns.DnsServiceClient
}

func MakeRecordsDeleter(dnsSVC dns.DnsServiceClient) RecordsDeleter {
	return RecordsDeleter{
		dnsSVC: dnsSVC,
	}
}

func MakeDefaultRecordsDeleter() (RecordsDeleter, error) {
	svc, err := dns.GetExternalDnsServiceClient()
	return MakeRecordsDeleter(svc), emperror.Wrap(err, "failed to get external DNS service client")
}

func (d RecordsDeleter) Delete(organizationID uint, clusterUID string) error {
	if d.dnsSVC == nil {
		return nil
	}

	if err := d.dnsSVC.DeleteDnsRecordsOwnedBy(clusterUID, organizationID); err != nil {
		return emperror.Wrapf(err, "deleting DNS records owned by cluster failed")
	}

	return nil
}
