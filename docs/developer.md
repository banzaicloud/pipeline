##Developer Guide

### How to run Pipeline in your local dev environment:

#### Prerequisites:

* Docker
* Account on Github

#### Pipeline dependencies 

``` bash
docker-compose -f docker-compose.yml up -d
``` 

This will create a `mysql`, `adminer` and `vault` container:
 - Adminer MySQL GUI: http://localhost:8080 login with username/password `sparky/sparky123`
 - Vault GUI: http://localhost:8200 login with token found in `cat ~/.vault-token`

#### Create your config.toml

Create a `config.toml` based on `config.toml.example`. These files must be placed under the `config` dir.
As of now the example config enables OAuth2 based authentication, and disables Drone deployment.
It can be changed by rewriting the example.

OAuth2 based authentication requires GitHub application, this can be created by following this 
[tutorial](https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/).
Please set the `clientid` and the `clientsecret` in the auth section, with the GitHub generated values.

> If you are not using HTTPS set auth.secureCookie = false, otherwise you won't be able to login via HTTP.

#### Set Required Environment Variables

For accessing Vault the `VAULT_ADDR` env var has to be set, Pipeline stores JWT access tokens there.

```bash
export VAULT_ADDR=http://127.0.0.1:8200
```

#### Route53 credentials in Vault

Organizations created in the Pipeline will have a domain registered in AWS's Route53 DNS Service. For this
the AWS credentials have to be available in Vault in the proper format (otherwise the feature is disabled):

```bash
vault kv put secret/banzaicloud/aws \
    AWS_REGION=${AWS_REGION} \
    AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
    AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
```

#### EKS cluster authentication

Creating and using EKS clusters requires to you to have the [AWS IAM Authenticator for Kubernetes](https://github.com/kubernetes-sigs/aws-iam-authenticator) installed on your machine:

```bash
go get github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
```

#### GitHub OAuth App setup

Setup your Pipeline GitHub OAuth application according to: [this guilde](./github-app.md)
