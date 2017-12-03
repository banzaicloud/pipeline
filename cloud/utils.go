package cloud

import (

	"github.com/kris-nova/kubicorn/state"
	"github.com/kris-nova/kubicorn/state/fs"

	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	notify "github.com/banzaicloud/pipeline/notify"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/logger"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/client-go/tools/clientcmd"
)

//We return stateStore so update can use it.
func getStateStoreForCluster(clusterType ClusterType) (stateStore state.ClusterStorer) {

	stateStore = fs.NewFileSystemStore(&fs.FileSystemStoreOptions{
		BasePath:    "statestore",
		ClusterName: clusterType.Name,
	})
	return stateStore
}

func assertTcpSocketAcceptsConnection(addr string) (bool, error) {
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

func expand(path string) string {
	if strings.Contains(path, "~") {
		return strings.Replace(path, "~", home(), 1)
	}
	return path
}

//GetConfig retrieves K8S config
func GetConfig(existing *cluster.Cluster, localDir string) (string, error) {
	user := existing.SSH.User
	pubKeyPath := expand(existing.SSH.PublicKeyPath)
	privKeyPath := strings.Replace(pubKeyPath, ".pub", "", 1)
	address := fmt.Sprintf("%s:%s", existing.KubernetesAPI.Endpoint, "22")
	if localDir == "" {
		localDir = fmt.Sprintf("./statestore/%s/", existing.Name)
	}
	localPath, err := getKubeConfigPath(localDir)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}
	sshConfig := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	remotePath := ""
	if user == "root" {
		remotePath = "/root/.kube/config"
	} else {
		remotePath = fmt.Sprintf("/home/%s/.kube/config", user)
	}

	agent := sshAgent()
	if agent != nil {
		auths := []ssh.AuthMethod{
			agent,
		}
		sshConfig.Auth = auths
	} else {
		pemBytes, err := ioutil.ReadFile(privKeyPath)
		if err != nil {

			return "", err
		}

		signer, err := getSigner(pemBytes)
		if err != nil {
			return "", err
		}

		auths := []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
		sshConfig.Auth = auths
	}

	sshConfig.SetDefaults()

	conn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	c, err := sftp.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer c.Close()
	r, err := c.Open(remotePath)
	if err != nil {
		return "", err
	}
	defer r.Close()
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}


	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		empty := []byte("")
		err := ioutil.WriteFile(localPath, empty, 0755)
		if err != nil {
			return "", err
		}
	}

	f, err := os.OpenFile(localPath, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(string(bytes))
	if err != nil {
		return "", err
	}
	defer f.Close()
	logger.Always("Wrote kubeconfig to [%s]", localPath)
	//TODO better solution
	config, err := clientcmd.BuildConfigFromFlags("", localPath)
	ioutil.WriteFile(localDir + "/client-key-data.pem", config.KeyData, 0644)
	ioutil.WriteFile(localDir + "/client-certificate-data.pem", config.CertData, 0644)
	ioutil.WriteFile(localDir + "/certificate-authority-data.pem", config.CAData, 0644)
	return localPath, nil
}

const (
	retryAttempts     = 150
	retrySleepSeconds = 5
)

//RetryGetConfig is retrying K8S config retrieval
func RetryGetConfig(existing *cluster.Cluster, localDir string) (string, error) {
	for i := 0; i <= retryAttempts; i++ {
		path, err := GetConfig(existing, localDir)
		if err != nil {
			if strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "getsockopt: connection refused") || strings.Contains(err.Error(), "unable to authenticate") {
				logger.Debug("Waiting for Kubernetes to come up..")
				time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
				continue
			}
			return "", err
		}
		notify.SlackNotify(fmt.Sprintf("Cluster Created: %s\n IP: %s", existing.Name, existing.KubernetesAPI.Endpoint))
		return path, err
	}
	return "", fmt.Errorf("Timeout writing kubeconfig")
}

func getKubeConfigPath(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s/config", path), nil
}

func sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func getSigner(pemBytes []byte) (ssh.Signer, error) {
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
