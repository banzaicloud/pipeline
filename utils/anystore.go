package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/ghodss/yaml"
)

type GenericFileSystemStoreOptions struct {
	AbsolutePath string
}

type GenericFileSystemStore struct {
	options      *GenericFileSystemStoreOptions
	AbsolutePath string
}

func NewGenericFileSystemStore(o *GenericFileSystemStoreOptions) *GenericFileSystemStore {
	return &GenericFileSystemStore{
		options:      o,
		AbsolutePath: o.AbsolutePath,
	}
}

func (fs *GenericFileSystemStore) Exists() bool {
	if _, err := os.Stat(fs.AbsolutePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (fs *GenericFileSystemStore) write(relativePath string, data []byte) error {
	fqn := fmt.Sprintf("%s/%s", fs.AbsolutePath, relativePath)
	os.MkdirAll(path.Dir(fqn), 0700)
	fo, err := os.Create(fqn)
	if err != nil {
		return err
	}
	defer fo.Close()
	_, err = io.Copy(fo, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	return nil
}

func (fs *GenericFileSystemStore) Read(relativePath string) ([]byte, error) {
	fqn := fmt.Sprintf("%s/%s", fs.AbsolutePath, relativePath)
	bytes, err := ioutil.ReadFile(fqn)
	if err != nil {
		return []byte(""), err
	}
	return bytes, nil
}

func (fs *GenericFileSystemStore) ReadStore(fileName string) ([]byte, error) {
	return fs.Read(fileName)
}

func (fs *GenericFileSystemStore) Commit(fileName string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("Nil data")
	}
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	fs.write(fileName, bytes)
	return nil
}

func (fs *GenericFileSystemStore) Rename(existingRelativePath, newRelativePath string) error {
	return os.Rename(existingRelativePath, newRelativePath)
}

func (fs *GenericFileSystemStore) Destroy() error {
	fmt.Printf("Removing path [%s]\n", fs.AbsolutePath)
	return os.RemoveAll(fs.AbsolutePath)
}

//func (fs *GenericFileSystemStore) GetCluster() (*cluster.Cluster, error) {
//	configBytes, err := fs.Read(state.ClusterYamlFile)
//	if err != nil {
//		return nil, err
//	}
//
//	return fs.BytesToCluster(configBytes)
//}

func (fs *GenericFileSystemStore) BytesToType(bytes []byte, outputType interface{}) (interface{}, error) {
	err := yaml.Unmarshal(bytes, outputType)
	if err != nil {
		return outputType, err
	}
	return outputType, nil
}

func (fs *GenericFileSystemStore) List() ([]string, error) {

	var stateList []string

	files, err := ioutil.ReadDir(fs.options.AbsolutePath)
	if err != nil {
		return stateList, err
	}

	for _, file := range files {
		stateList = append(stateList, file.Name())
	}

	return stateList, nil
}
