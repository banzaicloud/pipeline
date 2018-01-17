package utils

import (
	"github.com/kris-nova/kubicorn/state"
	"github.com/kris-nova/kubicorn/state/fs"

	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	"github.com/gin-gonic/gin"
	"github.com/kris-nova/kubicorn/cutil/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	JsonKeyStatus      = "status"
	JsonKeyMessage     = "message"
	JsonKeyName        = "name"
	JsonKeyError       = "error"
	JsonKeyResourceId  = "resourceId"
	JsonKeyIp          = "Ip"
	JsonKeyData        = "data"
	JsonKeyAvailable   = "available"
	JsonKeyAuth0       = "Auth0"
	JsonKeyReleaseName = "release_name"
	JsonKeyUrl         = "url"
	JsonKeyNotes       = "notes"
)

//We return stateStore so update can use it.
func GetStateStoreForCluster(clusterType banzaiSimpleTypes.ClusterSimple) (stateStore state.ClusterStorer) {

	stateStore = fs.NewFileSystemStore(&fs.FileSystemStoreOptions{
		BasePath:    "statestore",
		ClusterName: clusterType.Name,
	})
	return stateStore
}

func AssertTcpSocketAcceptsConnection(addr string) (bool, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false, fmt.Errorf("Attempting to open a socket to the Kubernetes API: %s", addr)
	}
	defer conn.Close()
	return true, nil
}

func home() string {
	home := os.Getenv("HOME")
	return home
}

func Expand(path string) string {
	if strings.Contains(path, "~") {
		return strings.Replace(path, "~", home(), 1)
	}
	return path
}

//GetEnv retrieves ENV variable, fallback if not set
func GetEnv(envKey, defaultValue string) string {
	value, exists := os.LookupEnv(envKey)
	if !exists {
		value = defaultValue
	}
	return value
}

//GetHomeDir retrieves Home on Linux
func GetHomeDir() string {
	//Linux
	return os.Getenv("HOME")
}

func GetKubeConfigPath(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s/config", path), nil
}

func GetSigner(pemBytes []byte) (ssh.Signer, error) {
	signerwithoutpassphrase, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		logger.Debug(err.Error())
		fmt.Print("SSH Key Passphrase [none]: ")
		passPhrase, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println("")
		if err != nil {
			return nil, err
		}
		signerwithpassphrase, err := ssh.ParsePrivateKeyWithPassphrase(pemBytes, passPhrase)
		if err != nil {
			return nil, err
		}

		return signerwithpassphrase, err
	}

	return signerwithoutpassphrase, err
}

//SetResponseBodyJson
func SetResponseBodyJson(c *gin.Context, statusCode int, obj interface{}) {
	if c != nil {
		c.JSON(statusCode, obj)
	}
}

//SetResponseBodyString
func SetResponseBodyString(c *gin.Context, statusCode int, format string, values... interface{}) {
	if c!= nil {
		c.String(statusCode, format, values)
	}
}

const (
	BootstrapScriptMasterKey     = "BOOTSTRAP_SCRIPT_MASTER"
	BootstrapScriptNodeKey       = "BOOTSTRAP_SCRIPT_NODE"
	BootstrapScriptMasterDefault = "https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_master_pipeline.sh"
	BootstrapScriptNodeDefault   = "https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_node_pipeline.sh"
)

func GetBootstrapScriptFromEnv(isMaster bool) string {

	var s string
	if isMaster {
		s = os.Getenv(BootstrapScriptMasterKey)
	} else {
		s = os.Getenv(BootstrapScriptNodeKey)
	}

	if len(s) == 0 {
		if isMaster {
			return BootstrapScriptMasterDefault
		} else {
			return BootstrapScriptNodeDefault
		}
	} else {
		return s
	}

}
