// Copyright © 2018 Banzai Cloud
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

package global

import (
	"path/filepath"
	"sync"

	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/pkg/crypto/cert"
	"github.com/banzaicloud/pipeline/secret"
)

// nolint: gochecknoglobals
var certGenerator *cert.Generator

// nolint: gochecknoglobals
var certGeneratorOnce sync.Once

func newCertGenerator() *cert.Generator {
	var caLoader cert.CALoader

	switch viper.GetString("cert.source") {
	case "file":
		caLoader = cert.NewFileCALoader(
			filepath.Join(viper.GetString("cert.path"), "ca.crt.pem"),
			filepath.Join(viper.GetString("cert.path"), "ca.key.pem"),
		)

	case "vault":
		caLoader = cert.NewVaultCALoader(secret.Store.Logical, viper.GetString("cert.path"))
	}

	generator := cert.NewGenerator(cert.NewCACache(caLoader))

	return generator
}

// GetCertGenerator returns the global cert generator instance.
func GetCertGenerator() *cert.Generator {
	certGeneratorOnce.Do(func() { certGenerator = newCertGenerator() })

	return certGenerator
}
