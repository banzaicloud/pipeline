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

package auditlog

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Clock provides time.
type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type realClock struct{}

func (realClock) Now() time.Time                  { return time.Now() }
func (realClock) Since(t time.Time) time.Duration { return time.Since(t) }

// Driver saves audit log entries.
type Driver interface {
	// Store saves an audit log entry.
	Store(entry Entry) error
}

// Option configures an audit log middleware.
type Option interface {
	// apply is unexported,
	// so only the current package can implement this interface.
	apply(o *middlewareOptions)
}

// In the future, sensitivePaths and userIDExtractor might be replaced by request matchers and propagators/decorators
// respectively to generalize them for multiple use cases, but for now this solution (borrowed from the previous one)
// should be fine.
type middlewareOptions struct {
	clock           Clock
	sensitivePaths  []*regexp.Regexp
	userIDExtractor func(req *http.Request) uint
	errorHandler    ErrorHandler
}

type optionFunc func(o *middlewareOptions)

func (fn optionFunc) apply(o *middlewareOptions) {
	fn(o)
}

// WithClock sets the clock in an audit log middleware.
func WithClock(clock Clock) Option {
	return optionFunc(func(o *middlewareOptions) {
		o.clock = clock
	})
}

// WithSensitivePaths marks API call paths as sensitive, causing the log entry to omit the request body.
func WithSensitivePaths(sensitivePaths []*regexp.Regexp) Option {
	return optionFunc(func(o *middlewareOptions) {
		o.sensitivePaths = sensitivePaths
	})
}

// WithErrorHandler sets the clock in an audit log middleware.
func WithErrorHandler(errorHandler ErrorHandler) Option {
	return optionFunc(func(o *middlewareOptions) {
		o.errorHandler = errorHandler
	})
}

// WithUserIDExtractor sets the function that extracts the user ID from the request.
func WithUserIDExtractor(userIDExtractor func(req *http.Request) uint) Option {
	return optionFunc(func(o *middlewareOptions) {
		o.userIDExtractor = userIDExtractor
	})
}

// Middleware returns a new HTTP middleware that records audit log entries.
func Middleware(driver Driver, opts ...Option) gin.HandlerFunc {
	options := middlewareOptions{
		clock:           realClock{},
		userIDExtractor: func(req *http.Request) uint { return 0 },
		errorHandler:    NoopErrorHandler{},
	}

	for _, opt := range opts {
		opt.apply(&options)
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		entry := Entry{
			Time:          options.clock.Now(),
			CorrelationID: c.GetString(correlationid.ContextKey),
			HTTP: HTTPEntry{
				ClientIP:  c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Method:    c.Request.Method,
				Path:      path,
			},
		}

		var sensitiveCall bool

		// Determine if this call contains sensitive information in its request body.
		for _, r := range options.sensitivePaths {
			if r.MatchString(c.Request.URL.Path) {
				sensitiveCall = true
				break
			}
		}

		// Only override the request body if there is actually one and it doesn't contain sensitive information.
		saveBody := c.Request.Body != nil && !sensitiveCall

		var buf bytes.Buffer

		if saveBody {
			// This should be ok, because the server keeps a reference to the original body,
			// so it can close the original request itself.
			c.Request.Body = ioutil.NopCloser(io.TeeReader(c.Request.Body, &buf))
		}

		c.Next() // process request

		entry.UserID = options.userIDExtractor(c.Request)

		// Consider making this configurable if you need to log unauthorized requests,
		// but keep in mind that in case of a public installation it's a potential DoS attack vector.
		if c.Writer.Status() == http.StatusUnauthorized {
			return
		}

		entry.HTTP.StatusCode = c.Writer.Status()
		entry.HTTP.ResponseSize = c.Writer.Size()
		entry.HTTP.ResponseTime = int(options.clock.Since(entry.Time).Milliseconds())

		if saveBody {
			// Make sure everything is read from the body.
			_, err := ioutil.ReadAll(c.Request.Body)
			if err != nil && err != io.EOF {
				options.errorHandler.HandleContext(c.Request.Context(), errors.WithStack(err))
			}

			entry.HTTP.RequestBody = string(buf.Bytes())
		}

		if c.IsAborted() {
			for _, e := range c.Errors {
				_e, _ := e.MarshalJSON()

				entry.HTTP.Errors = append(entry.HTTP.Errors, string(_e))
			}
		}

		err := driver.Store(entry)
		if err != nil {
			options.errorHandler.HandleContext(c.Request.Context(), errors.WithStackIf(err))
		}
	}
}
