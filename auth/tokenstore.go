package auth

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"sync"

	"k8s.io/client-go/rest"

	vault "github.com/hashicorp/vault/api"
)

// TokenStore is general interface for storing access tokens
type TokenStore interface {
	Store(string, string) error
	Lookup(string, string) (bool, error)
	Revoke(string, string) error
	List(string) ([]string, error)
}

// In-memory implementation

// NewInMemoryTokenStore is a basic in-memory TokenStore implementation (thread-safe)
func NewInMemoryTokenStore() TokenStore {
	return &inMemoryTokenStore{store: make(map[string]map[string]bool)}
}

type inMemoryTokenStore struct {
	sync.RWMutex
	store map[string]map[string]bool
}

func (tokenStore *inMemoryTokenStore) Store(userId, token string) error {
	tokenStore.Lock()
	defer tokenStore.Unlock()
	var userTokens map[string]bool
	var ok bool
	if userTokens, ok = tokenStore.store[userId]; !ok {
		userTokens = make(map[string]bool)
	}
	userTokens[token] = true
	tokenStore.store[userId] = userTokens
	return nil
}

func (tokenStore *inMemoryTokenStore) Lookup(userId, token string) (bool, error) {
	tokenStore.RLock()
	defer tokenStore.RUnlock()
	if userTokens, ok := tokenStore.store[userId]; ok {
		_, found := userTokens[token]
		return found, nil
	}
	return false, nil
}

func (tokenStore *inMemoryTokenStore) Revoke(userId, token string) error {
	tokenStore.Lock()
	defer tokenStore.Unlock()
	if userTokens, ok := tokenStore.store[userId]; ok {
		delete(userTokens, token)
	}
	return nil
}

func (tokenStore *inMemoryTokenStore) List(userId string) ([]string, error) {
	tokenStore.Lock()
	defer tokenStore.Unlock()
	if userTokens, ok := tokenStore.store[userId]; ok {
		tokens := make([]string, len(userTokens))
		i := 0
		for k := range userTokens {
			tokens[i] = k
			i++
		}
		return tokens, nil
	}
	return nil, nil
}

// Vault based implementation

// A TokenStore implementation which stores tokens in Vault
// For local development:
// $ vault server -dev &
// $ export VAULT_ADDR='http://127.0.0.1:8200'
type vaultTokenStore struct {
	client       *vault.Client
	logical      *vault.Logical
	tokenRenewer *vault.Renewer
}

func NewVaultTokenStore() TokenStore {
	client, err := vault.NewClient(vault.DefaultConfig())
	if err != nil {
		panic(err)
	}
	logical := client.Logical()
	var tokenRenewer *vault.Renewer

	if client.Token() == "" {

		usr, err := user.Current()
		if err != nil {
			panic(err)
		}

		token, err := ioutil.ReadFile(usr.HomeDir + "/.vault-token")
		if err == nil {

			client.SetToken(string(token))

		} else {
			// If VAULT_TOKEN or ~/.vault-token wasn't provided let's suppose
			// we are in Kubernetes and try to get one with the ServiceAccount token

			k8sconfig, err := rest.InClusterConfig()
			if err != nil {
				panic(err)
			}

			data := map[string]interface{}{"jwt": k8sconfig.BearerToken, "role": "pipeline"}
			secret, err := logical.Write("auth/kubernetes/login", data)
			if err != nil {
				panic(err)
			}

			tokenRenewer, err = client.NewRenewer(&vault.RenewerInput{Secret: secret})
			if err != nil {
				panic(err)
			}

			// We never really want to stop this
			go tokenRenewer.Renew()

			// Finally set the first token from the response
			client.SetToken(secret.Auth.ClientToken)
		}
	}

	return vaultTokenStore{client: client, logical: logical, tokenRenewer: tokenRenewer}
}

func tokenPath(userId, token string) string {
	return fmt.Sprintf("secret/accesstokens/%s/%s", userId, token)
}

func (tokenStore vaultTokenStore) Store(userId, token string) error {
	data := map[string]interface{}{"token": token}
	_, err := tokenStore.logical.Write(tokenPath(userId, token), data)
	return err
}

func (tokenStore vaultTokenStore) Lookup(userId, token string) (bool, error) {
	secret, err := tokenStore.logical.Read(tokenPath(userId, token))
	if err != nil {
		return false, err
	}
	return secret != nil, nil
}

func (tokenStore vaultTokenStore) Revoke(userId, token string) error {
	_, err := tokenStore.logical.Delete(tokenPath(userId, token))
	return err
}

func (tokenStore vaultTokenStore) List(userId string) ([]string, error) {
	secret, err := tokenStore.logical.List(fmt.Sprintf("secret/accesstokens/%s", userId))
	if err != nil {
		return nil, err
	}

	keys := secret.Data["keys"].([]interface{})
	tokens := make([]string, len(keys))
	for i, key := range keys {
		tokens[i] = key.(string)
	}
	return tokens, nil
}
