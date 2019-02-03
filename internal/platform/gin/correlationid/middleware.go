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

package correlationid

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

// ContextKey is the key the retrieved (or generated) correlation ID is stored under in the gin Context.
const ContextKey = "correlationid"

// Default correlation ID header
const defaultHeader = "Correlation-ID"

// MiddlewareOption configures the correlation ID middleware.
type MiddlewareOption interface {
	apply(*middleware)
}

// Header configures the header from where the correlation ID will be retrieved.
type Header string

// apply implements the MiddlewareOption interface.
func (h Header) apply(m *middleware) {
	m.header = string(h)
}

// Middleware returns a gin compatible handler.
func Middleware(opts ...MiddlewareOption) gin.HandlerFunc {
	m := new(middleware)

	for _, opt := range opts {
		opt.apply(m)
	}

	if m.header == "" {
		m.header = defaultHeader
	}

	return m.Handle
}

type middleware struct {
	header string
}

func (m *middleware) Handle(ctx *gin.Context) {
	if header := ctx.GetHeader(m.header); header != "" {
		ctx.Set(ContextKey, header)
	} else {
		ctx.Set(ContextKey, uuid.Must(uuid.NewV4()).String())
	}

	ctx.Next()
}
