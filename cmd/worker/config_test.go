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

package main

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	var config Config

	v := viper.New()
	p := pflag.NewFlagSet("test", pflag.ContinueOnError)

	Configure(v, p)

	file, err := os.Open("../../config/config.toml.dist")
	require.NoError(t, err)

	v.SetConfigType("toml")

	err = v.ReadConfig(file)
	require.NoError(t, err)

	err = v.Unmarshal(&config)
	require.NoError(t, err)

	err = config.Validate()
	require.NoError(t, err)
}
