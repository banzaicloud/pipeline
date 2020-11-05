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

package cadence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/workflow"
)

type headerWriter map[string][]byte

func (w headerWriter) Set(s string, bytes []byte) {
	w[s] = bytes
}

type headerReader map[string][]byte

func (r headerReader) ForEachKey(handler func(string, []byte) error) error {
	for key, value := range r {
		if err := handler(key, value); err != nil {
			return err
		}
	}

	return nil
}

type contextKey string

const (
	propagationKey  = "_prop"
	dummyContextKey = contextKey("dummy")
	contextValue    = "value"
)

func TestStringContextPropagator_Inject(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), dummyContextKey, contextValue)

		headerWriter := headerWriter{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		err := cp.Inject(ctx, headerWriter)
		require.NoError(t, err)

		assert.Equal(t, contextValue, string(headerWriter[propagationKey]))
	})

	t.Run("Optional", func(t *testing.T) {
		ctx := context.Background()

		headerWriter := headerWriter{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
			Optional:       true,
		}

		err := cp.Inject(ctx, headerWriter)
		require.NoError(t, err)

		assert.Len(t, headerWriter, 0)
	})

	t.Run("Missing", func(t *testing.T) {
		ctx := context.Background()

		headerWriter := headerWriter{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		err := cp.Inject(ctx, headerWriter)
		require.Error(t, err)
	})
}

func TestStringContextPropagator_InjectFromWorkflow(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ctx := workflow.WithValue(new(emptyCtx), dummyContextKey, contextValue)

		headerWriter := headerWriter{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		err := cp.InjectFromWorkflow(ctx, headerWriter)
		require.NoError(t, err)

		assert.Equal(t, contextValue, string(headerWriter[propagationKey]))
	})

	t.Run("Optional", func(t *testing.T) {
		ctx := new(emptyCtx)

		headerWriter := headerWriter{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
			Optional:       true,
		}

		err := cp.InjectFromWorkflow(ctx, headerWriter)
		require.NoError(t, err)

		assert.Len(t, headerWriter, 0)
	})

	t.Run("Missing", func(t *testing.T) {
		ctx := new(emptyCtx)

		headerWriter := headerWriter{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		err := cp.InjectFromWorkflow(ctx, headerWriter)
		require.Error(t, err)
	})
}

func TestStringContextPropagator_Extract(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		headerReader := headerReader{
			propagationKey: []byte(contextValue),
		}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		ctx, err := cp.Extract(context.Background(), headerReader)
		require.NoError(t, err)

		val, ok := ctx.Value(dummyContextKey).(string)
		require.True(t, ok)

		assert.Equal(t, contextValue, val)
	})

	t.Run("Missing", func(t *testing.T) {
		headerReader := headerReader{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		ctx, err := cp.Extract(context.Background(), headerReader)
		require.NoError(t, err)

		_, ok := ctx.Value(dummyContextKey).(string)
		require.False(t, ok)
	})
}

func TestStringContextPropagator_ExtractToWorkflow(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		headerReader := headerReader{
			propagationKey: []byte(contextValue),
		}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		ctx, err := cp.ExtractToWorkflow(new(emptyCtx), headerReader)
		require.NoError(t, err)

		val, ok := ctx.Value(dummyContextKey).(string)
		require.True(t, ok)

		assert.Equal(t, contextValue, val)
	})

	t.Run("Missing", func(t *testing.T) {
		headerReader := headerReader{}

		cp := StringContextPropagator{
			PropagationKey: propagationKey,
			ContextKey:     dummyContextKey,
		}

		ctx, err := cp.ExtractToWorkflow(new(emptyCtx), headerReader)
		require.NoError(t, err)

		_, ok := ctx.Value(dummyContextKey).(string)
		require.False(t, ok)
	})
}
