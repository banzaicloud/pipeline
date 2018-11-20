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
	"crypto/rand"
	"path/filepath"
	"sync"

	"github.com/banzaicloud/pipeline/pkg/crypto/cert"
	"github.com/spf13/viper"
)

var certGenerator *cert.Generator
var certGeneratorOnce sync.Once

func newCertGenerator() *cert.Generator {
	generator, err := cert.NewGeneratorFromFile(
		filepath.Join(viper.GetString("cert.path"), "ca.crt.pem"),
		filepath.Join(viper.GetString("cert.path"), "ca.key.pem"),
		cert.SystemClock,
		rand.Reader,
	)
	if err != nil {
		panic(err)
	}

	return generator
}

// GetCertGenerator returns the global cert generator instance.
func GetCertGenerator() *cert.Generator {
	certGeneratorOnce.Do(func() { certGenerator = newCertGenerator() })

	return certGenerator
}
