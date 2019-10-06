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

package anchore

import (
	"context"
	"testing"

	"github.com/banzaicloud/pipeline/.gen/anchore"
)

func TestMakeAnchoreClient(t *testing.T) {

	anchoreCli := anchore.NewAPIClient(&anchore.Configuration{
		BasePath:      "https://alpha.dev.banzaicloud.com/imagecheck",
		DefaultHeader: make(map[string]string),
		UserAgent:     "Pipeline/go",
	})

	//s, r, e := anchoreCli.DefaultApi.Ping(context.Background())

	auth := context.WithValue(context.Background(), anchore.ContextBasicAuth, anchore.BasicAuth{
		UserName: "admin",
		Password: "3jpSQH8N6FSM",
	})

	s, r, e := anchoreCli.UserManagementApi.ListAccounts(auth, &anchore.ListAccountsOpts{})
	print(s, r, e)

}
