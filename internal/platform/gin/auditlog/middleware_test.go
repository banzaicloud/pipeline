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
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

type inmemDriver struct {
	entries []Entry

	mu sync.Mutex
}

func (d *inmemDriver) Store(entry Entry) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.entries = append(d.entries, entry)

	return nil
}

func TestMiddleware(t *testing.T) {
	t.Run("LogsAPICall", func(t *testing.T) {
		now := time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC)
		clock := clockwork.NewFakeClockAt(now)

		userIDExtractor := func(req *http.Request) uint { return 1 }

		body := "Hello, World!"

		entry := Entry{
			Time:          now,
			CorrelationID: "cid",
			UserID:        1,
			HTTP: HTTPEntry{
				ClientIP:     "127.0.0.1",
				UserAgent:    "go-test",
				Method:       http.MethodPost,
				Path:         "/?a=b",
				RequestBody:  body,
				StatusCode:   http.StatusOK,
				ResponseTime: 1000,
				ResponseSize: 18,
				Errors:       nil,
			},
		}

		driver := &inmemDriver{}

		middleware := Middleware(driver, WithClock(clock), WithUserIDExtractor(userIDExtractor))

		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set(correlationid.ContextKey, "cid") }, middleware)
		engine.POST("/", func(c *gin.Context) {
			clock.Advance(time.Second)

			c.JSON(http.StatusOK, map[string]string{"hello": "world"})
		})

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/?a=b", bytes.NewReader([]byte("Hello, World!")))
		require.NoError(t, err)

		req.Header.Set("User-Agent", "go-test")
		req.Header.Set("X-Real-Ip", "127.0.0.1")

		engine.ServeHTTP(w, req)

		assert.Equal(t, entry, driver.entries[0])
	})

	t.Run("FiltersSensitiveInformation", func(t *testing.T) {
		driver := &inmemDriver{}

		middleware := Middleware(driver, WithSensitivePaths([]*regexp.Regexp{regexp.MustCompile("^/sensitive-path$")}))

		engine := gin.New()
		engine.Use(middleware)
		engine.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })
		engine.POST("/sensitive-path", func(c *gin.Context) { c.Status(http.StatusOK) })

		{
			w := httptest.NewRecorder()
			req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte("Hello, World!")))
			require.NoError(t, err)

			engine.ServeHTTP(w, req)
		}

		{
			w := httptest.NewRecorder()
			req, err := http.NewRequest("POST", "/sensitive-path", bytes.NewReader([]byte("Hello, Secret World!")))
			require.NoError(t, err)

			engine.ServeHTTP(w, req)
		}

		assert.Equal(t, "Hello, World!", driver.entries[0].HTTP.RequestBody)
		assert.Empty(t, driver.entries[1].HTTP.RequestBody)
	})

	t.Run("SkipsUnauthorizedRequests", func(t *testing.T) {
		driver := &inmemDriver{}

		middleware := Middleware(driver)

		engine := gin.New()
		engine.Use(middleware)
		engine.POST("/", func(c *gin.Context) { c.Status(http.StatusUnauthorized) })

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/", nil)
		require.NoError(t, err)

		engine.ServeHTTP(w, req)

		assert.Len(t, driver.entries, 0)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		driver := &inmemDriver{}

		middleware := Middleware(driver)

		engine := gin.New()
		engine.Use(middleware)
		engine.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/", nil)
		require.NoError(t, err)

		engine.ServeHTTP(w, req)

		assert.Empty(t, driver.entries[0].HTTP.RequestBody)
	})

	t.Run("LogsAbortedRequests", func(t *testing.T) {
		driver := &inmemDriver{}

		middleware := Middleware(driver)

		engine := gin.New()
		engine.Use(middleware)
		engine.POST("/", func(c *gin.Context) {
			_ = c.AbortWithError(http.StatusBadRequest, errors.New("invalid request"))
		})

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/", nil)
		require.NoError(t, err)

		engine.ServeHTTP(w, req)

		assert.Equal(t, []string{"{\"error\":\"invalid request\"}"}, driver.entries[0].HTTP.Errors)
	})
}
