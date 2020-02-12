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
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/hook"
	"github.com/banzaicloud/pipeline/pkg/mirror"
)

func TestConfigure(t *testing.T) {
	var config configuration

	v := viper.NewWithOptions(
		viper.KeyDelimiter("::"),
	)
	p := pflag.NewFlagSet("test", pflag.ContinueOnError)

	configure(v, p)

	file, err := os.Open("../../config/config.dev.yaml")
	require.NoError(t, err)

	v.SetConfigType("yaml")

	err = v.ReadConfig(file)
	require.NoError(t, err)

	err = v.Unmarshal(&config, hook.DecodeHookWithDefaults())
	require.NoError(t, err)

	err = config.Process()
	require.NoError(t, err)

	err = config.Validate()
	require.NoError(t, err)
}

func TestConfigureForUsedDefaults(t *testing.T) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("::"),
	)
	p := pflag.NewFlagSet("test", pflag.ContinueOnError)

	configure(v, p)

	WithErrorUnused := viper.DecoderConfigOption(func(cfg *mapstructure.DecoderConfig) {
		cfg.ErrorUnused = true
	})

	var config configuration
	err := v.Unmarshal(&config, hook.DecodeHookWithDefaults(), WithErrorUnused)
	require.NoError(t, err)
}

func TestGlobalConfigCoverage(t *testing.T) {
	globalConfigType := reflect.TypeOf(global.Config)
	gc1 := reflect.New(globalConfigType).Elem()
	gc2 := reflect.New(globalConfigType)

	fillWithNonZeroValue(t, gc1)

	var cfg configuration
	err := mapstructure.Decode(gc1.Interface(), &cfg)
	require.NoError(t, err)

	err = mapstructure.Decode(cfg, gc2.Interface())
	require.NoError(t, err)

	require.Equal(t, gc1.Interface(), gc2.Elem().Interface())
}

func fillWithNonZeroValue(t *testing.T, v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(42)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(42)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(42)
	case reflect.Complex64, reflect.Complex128:
		v.SetComplex(42 + 42i)
	case reflect.Array:
		l := v.Type().Len()
		for i := 0; i < l; i++ {
			fillWithNonZeroValue(t, v.Index(i))
		}
	case reflect.Interface:
		// leave as nil
	case reflect.Map:
		vt := v.Type()
		if v.IsNil() {
			v.Set(reflect.MakeMap(vt))
		}
		key := reflect.New(vt.Key()).Elem()
		val := reflect.New(vt.Elem()).Elem()
		fillWithNonZeroValue(t, key)
		fillWithNonZeroValue(t, val)
		v.SetMapIndex(key, val)
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
	case reflect.Slice:
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		}
		l := v.Len()
		for i := 0; i < l; i++ {
			fillWithNonZeroValue(t, v.Index(i))
		}
	case reflect.String:
		v.SetString("lorem")
	case reflect.Struct:
		it := mirror.NewStructIter(v)
		for it.Next() {
			fillWithNonZeroValue(t, it.Value())
		}
	default:
		t.Fatalf("unhandled type in global config: %v", v.Kind())
	}
}
