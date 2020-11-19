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

package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func helloHandler(c *gin.Context) {
	user := c.Request.Context().Value(CurrentUser)
	if user != nil {
		c.String(http.StatusOK, "hello user")
	} else {
		c.AbortWithStatus(http.StatusForbidden)
	}
}

func newServer(t *testing.T) *httptest.Server {
	internalHandler := newInternalHandler(NewServiceAccountService())

	router := gin.Default()
	router.Use(internalHandler)
	router.GET("/hello", helloHandler)

	server := httptest.NewUnstartedServer(router)

	tlsConfig, err := TLSConfigForClientAuth("../../config/certs/ca.pem")
	if err != nil {
		t.Fatal(err)
	}

	server.TLS = tlsConfig
	server.StartTLS()

	return server
}

func testInternalHandlerWithoutClientCertificate(t *testing.T) {
	server := newServer(t)
	defer server.Close()

	client := server.Client()
	resp, err := client.Get(server.URL + "/hello")
	require.NoError(t, err)

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func testInternalHandlerWithClientCertificate(t *testing.T) {
	server := newServer(t)
	defer server.Close()

	client := server.Client()
	clientTransport := client.Transport.(*http.Transport)

	clientCert, err := tls.LoadX509KeyPair("../../config/certs/client.pem", "../../config/certs/client-key.pem")
	if err != nil {
		t.Fatal(err, "failed to load TLS client certificate")
	}

	clientTransport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}

	resp, err := client.Get(server.URL + "/hello")
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func testInternalHandlerWithBadClientCertificate(t *testing.T) {
	server := newServer(t)
	defer server.Close()

	client := server.Client()
	clientTransport := client.Transport.(*http.Transport)

	clientCert, err := tls.LoadX509KeyPair("../../config/certs/client-non-pipeline.pem", "../../config/certs/client-non-pipeline-key.pem")
	if err != nil {
		t.Fatal(err, "failed to load TLS client certificate")
	}

	clientTransport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}

	_, err = client.Get(server.URL + "/hello")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote error: tls: bad certificate")
}
