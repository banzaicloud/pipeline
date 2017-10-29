package test

import (
	"fmt"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/logger"
	"github.com/kris-nova/kubicorn/profiles"
	"github.com/kris-nova/kubicorn/test"
	"net"
	"os"
	"testing"
	"time"
)

var testCluster *cluster.Cluster

func TestMain(m *testing.M) {
	logger.TestMode = true
	logger.Level = 4
	var err error

	testCluster = profiles.NewSimpleAmazonCluster("aws-ubuntu-test")
}

const (
	ApiSleepSeconds   = 5
	ApiSocketAttempts = 40
)

func TestApiListen(t *testing.T) {
	success := false
	for i := 0; i < ApiSocketAttempts; i++ {
		_, err := assertTcpSocketAcceptsConnection(fmt.Sprintf("%s:%s", testCluster.KubernetesApi.Endpoint, testCluster.KubernetesApi.Port), "opening a new socket connection against the Kubernetes API")
		if err != nil {
			logger.Info("Attempting to open a socket to the Kubernetes API: %v...\n", err)
			time.Sleep(time.Duration(ApiSleepSeconds) * time.Second)
			continue
		}
		success = true
	}
	if !success {
		t.Fatalf("Unable to connect to Kubernetes API")
	}
}

func assertTcpSocketAcceptsConnection(addr, msg string) (bool, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false, fmt.Errorf("%s: %s", msg, err)
	}
	defer conn.Close()
	return true, nil
}
