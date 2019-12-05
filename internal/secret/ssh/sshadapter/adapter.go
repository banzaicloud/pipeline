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

package sshadapter

import (
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/src/secret"
)

func KeyPairFromSecret(s *secret.SecretItemResponse) ssh.KeyPair {
	return ssh.KeyPair{
		User:                 s.Values[secrettype.User],
		Identifier:           s.Values[secrettype.Identifier],
		PublicKeyData:        s.Values[secrettype.PublicKeyData],
		PublicKeyFingerprint: s.Values[secrettype.PublicKeyFingerprint],
		PrivateKeyData:       s.Values[secrettype.PrivateKeyData],
	}
}
