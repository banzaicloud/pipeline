##Developer Guide

### How to run Pipeline in your local dev environment:

#### Prerequisites:

* Docker
* Account on Github

#### Pipeline dependencies 

``` bash
docker-compose -f docker-compose-local.yml up -d
``` 

This will create a `mysql`, `adminer` and `vault` container.

#### Create your config.toml

Create a `config.toml` based on `config.toml.example`. These files must be placed under the `config` dir.
As of now the example config enables OAuth2 based authentication, and disables Drone deployment.
It can be changed by rewriting the example.

OAuth2 based authentication requires GitHub application, this can be created by following this 
[tutorial](https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/).
Please set the `clientid` and the `clientsecret` in the auth section, with the GitHub generated values.

By default Pipeline uses public/private key from `~/.ssh/id_rsa.pub`. If this key is protected with
passphrase or the keys stored elsewhere, modify the config.toml to point towards the right key. This can be done
by modifying the `cloud` section `keypath` entry. This needs to point to the `public` key.

#### Set Required Environment Variables

For accessing Vault the `VAULT_ADDR` env var has to be set, Pipeline stores JWT access tokens there.

```bash
VAULT_ADDR=http://127.0.0.1:8200
```

Depending on the cloud provider there are couple of env vars has to be set:

* AKS
   * AZURE_CLIENT_ID
   * AZURE_CLIENT_SECRET
   * AZURE_TENANT_ID
   * AZURE_SUBSCRIPTION_ID
* Amazon
   * AWS_ACCESS_KEY_ID
   * AWS_SECRET_ACCESS_KEY
*GCP

#### GitHub OAuth App setup

Setup your Pipeline GitHub OAuth application according to: [this guilde](./github-app.md)
