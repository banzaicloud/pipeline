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

package viperx

import (
	"github.com/spf13/viper"
)

// RegisterAlias provides another accessor for the same key.
// It's useful for backward compatible configuration changes.
//
// Compared to the original RegisterAlias function, this one works on nested keys.
func RegisterAlias(v *viper.Viper, alias string, key string) {
	if v.IsSet(alias) {
		v.Set(key, v.Get(alias))
		v.RegisterAlias(alias, key)
	}
}
