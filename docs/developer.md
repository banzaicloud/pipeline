##Developer Guide

### How to run pipeline in your local dev environment:

#### Prerequisites:

* Docker
* Account on Github (if oauth enabled)

####You need to create the surrounding deps. To do that run:

``` bash
docker-compose -f docker-compose-local.yml up -d
``` 

This will create a mysql, adminer, vault container. The first two is always required by the pipeline,
vault is only required when oauth based authentication enabled.

#### Create your config.toml

Create a `config.toml` based on `config.toml.example`. These files must be placed under the `config` dir.
As of now the example config enables oauth based authentication, and disables drone deployment.
It can be changed by rewriting the example.

Oauth based authentication requires github application, this can be created by following this 
[tutorial](https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/).
Please set the `clientid` and the `clientsecret` in the auth section, with the github generated values.

By default pipeline uses public/private key from `~/.ssh/id_rsa.pub`. If this key is protected with
passphrase or the keys stored elsewhere, modify the config.toml to point the right key. This can be done
by modifying the `cloud` section `keypath` entry. This needs to point to the `public` key.

#### Set Required Environment Variables

If the oauth based authentication is enabled, the `VAULT_ADDR` env var has to be set.

```bash
VAULT_ADDR=http://127.0.0.1:8200
```

Despite of cloud provider, there are couple of env vars has to be set:

* AKS
   * AZURE_CLIENT_ID
   * AZURE_CLIENT_SECRET
   * AZURE_TENANT_ID
   * AZURE_SUBSCRIPTION_ID
* Amazon
   * AWS_ACCESS_KEY_ID
   * AWS_SECRET_ACCESS_KEY
*GCP

#### Run Pipeline

If the oauth based authentication is enabled, a token has to be generated,
the above created Github Oauth App `Authorization callback URL` section should be:

```bash
http://localhost:9090/auth/github/callback
```

Token can be generated only with browser, to do that please use the following URL:

```bash
localhost:9090/auth/github/login
```

Please authenticate yourself with github. If everything is done correctly you will be redirected.
The browser already contains the generated token. It can be retrieved by issuing the following:

```bash
http://localhost:9090/api/v1/token
```
