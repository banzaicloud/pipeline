package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

//NopHandler is an empty handler to help net/http -> Gin conversions
type NopHandler struct{}

func (h NopHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {}

//WriteToFile write the []byte to the given file
func WriteToFile(data []byte, file string) error {
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return ioutil.WriteFile(file, data, 0644)
	}

	tmpfi, err := ioutil.TempFile(filepath.Dir(file), "file.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfi.Name())

	if err = ioutil.WriteFile(tmpfi.Name(), data, 0644); err != nil {
		return err
	}

	if err = tmpfi.Close(); err != nil {
		return err
	}

	if err = os.Remove(file); err != nil {
		return err
	}

	err = os.Rename(tmpfi.Name(), file)
	return err
}

// ConvertJson2Map converts []byte to map[string]string
func ConvertJson2Map(js []byte) (map[string]string, error) {
	var result map[string]string
	err := json.Unmarshal(js, &result)
	return result, err
}

// Contains checks slice contains `s` string
func Contains(slice []string, s string) bool {
	for _, sl := range slice {
		if sl == s {
			return true
		}
	}
	return false
}

// EncodeStringToBase64 first checks if the string is encoded if yes returns it if no than encodes it.
func EncodeStringToBase64(s string) string {
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}
	return s
}

// ConvertSecondsToTime returns string format of seconds
func ConvertSecondsToTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// GetOrCreateClusterRole gets the cluster role with the given name if exists otherwise creates new one and returns it
func GetOrCreateClusterRole(log logrus.FieldLogger, client *kubernetes.Clientset, name string, rules []v1beta1.PolicyRule) (*v1beta1.ClusterRole, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{"metadata.name": name})

	clusterRoles, err := client.RbacV1beta1().ClusterRoles().List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		log.Errorf("querying cluster roles failed: %s", err.Error())
		return nil, err
	}

	if len(clusterRoles.Items) > 1 {
		log.Errorf("duplicate cluster role with name %q found", name)
		return nil, fmt.Errorf("duplicate cluster role with name %q found", name)
	}

	if len(clusterRoles.Items) == 1 {
		log.Infof("cluster role %q already exists", name)
		return &clusterRoles.Items[0], nil
	}

	clusterRole, err := client.RbacV1beta1().ClusterRoles().Create(
		&v1beta1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Rules: rules,
		})

	if err != nil {
		log.Errorf("creating cluster role %q failed: %s", name, err.Error())
		return nil, err
	}

	log.Infof("cluster role %q created", name)

	return clusterRole, nil
}

// GetOrCreateServiceAccount checks is service account with given name exists in the specified namespace and returns it.
// if it doesn't exists it creates a new one and returns it to the caller.
func GetOrCreateServiceAccount(log logrus.FieldLogger, client *kubernetes.Clientset, namespace, name string) (*v1.ServiceAccount, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{"metadata.name": name})

	serviceAccounts, err := client.CoreV1().ServiceAccounts(namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		log.Errorf("querying service accounts in namespace %q failed: %s", namespace, err.Error())
		return nil, err
	}

	if len(serviceAccounts.Items) > 1 {
		log.Errorf("duplicate service account with '%s/%s' found ", namespace, name)
		return nil, fmt.Errorf("duplicate service account with '%s/%s' found ", namespace, name)
	}

	if len(serviceAccounts.Items) == 1 {
		log.Infof("service account '%s/%s' already exists", namespace, name)
		return &serviceAccounts.Items[0], nil
	}

	serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Create(
		&v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})

	if err != nil {
		log.Errorf("creating service account '%s/%s' failed: %s", namespace, name, err.Error())
		return nil, err
	}

	log.Infof("service account '%s/%s' created", namespace, name)

	return serviceAccount, nil
}

// GetOrCreateClusterRoleBinding creates the cluster role binding given its name, service account and cluster role if not exists.
// It returns the found cluster role binding if one already exists or the newly created one.
func GetOrCreateClusterRoleBinding(log logrus.FieldLogger,
	client *kubernetes.Clientset,
	name string, serviceAccount *v1.ServiceAccount,
	clusterRole *v1beta1.ClusterRole) (*v1beta1.ClusterRoleBinding, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{"metadata.name": name})

	clusterRoleBindings, err := client.RbacV1beta1().ClusterRoleBindings().List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		log.Errorf("querying cluster role bindings failed: %s", err.Error())
		return nil, err
	}

	if len(clusterRoleBindings.Items) > 1 {
		log.Errorf("duplicate cluster role binding with name %q found", name)
		return nil, fmt.Errorf("duplicate cluster role binding with name %q found", name)
	}

	if len(clusterRoleBindings.Items) == 1 {
		log.Infof("cluster role binding %q already exists", name)
		return &clusterRoleBindings.Items[0], nil
	}

	clusterRoleBinding, err := client.RbacV1beta1().ClusterRoleBindings().Create(
		&v1beta1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Subjects: []v1beta1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccount.Name,
					Namespace: serviceAccount.Namespace,
					APIGroup:  v1.GroupName,
				},
			},
			RoleRef: v1beta1.RoleRef{
				Kind:     "ClusterRole",
				Name:     clusterRole.Name,
				APIGroup: v1beta1.GroupName,
			},
		})

	if err != nil {
		log.Errorf("creating cluster role binding %q failed: %s", name, err.Error())
		return nil, err
	}

	log.Infof("cluster role binding %q created", name)

	return clusterRoleBinding, nil
}
