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
	"errors"
	"strings"

	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/validation"
)

// GetBaseDomain returns the DNS base domain from [dns.domain] config. Changes the read domain to lowercase
// to ensure it's DNS-1123 compliant
func GetBaseDomain() (string, error) {
	baseDomain := strings.ToLower(viper.GetString(config.DNSBaseDomain))

	err := ValidateSubdomain(baseDomain)
	if err != nil {
		return "", emperror.WrapWith(err, "invalid base domain")
	}

	return baseDomain, nil
}

// ValidateSubdomain verifies if the provided subdomain string complies with DNS-1123
func ValidateSubdomain(subdomain string) error {
	errs := validation.IsDNS1123Subdomain(subdomain)
	if len(errs) > 0 {
		return emperror.With(errors.New(strings.Join(errs, "\n")), "subdomain", subdomain)
	}

	return nil
}

// ValidateWildcardSubdomain verifies if the provided subdomain string complies with wildcard DNS-1123
func ValidateWildcardSubdomain(subdomain string) error {
	errs := validation.IsWildcardDNS1123Subdomain(subdomain)
	if len(errs) > 0 {
		return emperror.With(errors.New(strings.Join(errs, "\n")), "subdomain", subdomain)
	}

	return nil
}
