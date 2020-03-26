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

package auth

import (
	"net/http"
)

const pipelineServiceAccountLogin = "pipeline"

type ServiceAccountService interface {
	ExtractServiceAccount(*http.Request) *User
	IsAdminServiceAccount(*User) bool
}

type serviceAccountService struct {
}

func NewServiceAccountService() ServiceAccountService {
	return serviceAccountService{}
}

func (s serviceAccountService) ExtractServiceAccount(r *http.Request) *User {
	if r.TLS == nil {
		return nil
	}

	if len(r.TLS.VerifiedChains) > 0 {
		user := User{
			ID:             0,
			Login:          pipelineServiceAccountLogin,
			ServiceAccount: true,
		}

		return &user
	}

	return nil
}

func (s serviceAccountService) IsAdminServiceAccount(u *User) bool {
	if u.ID == 0 && u.ServiceAccount {
		switch u.Login {
		case pipelineServiceAccountLogin:
			return true
		}
	}

	return false
}
