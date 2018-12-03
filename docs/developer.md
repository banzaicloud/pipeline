


# Developer Guide

## How to run Pipeline in your local dev environment

### Prerequisites

- Make
- Docker (with Compose)
- Account on Github


### GitHub OAuth App setup

Setup your Pipeline GitHub OAuth application according to [this guide](./github-app.md)


### Quick start

To spin up the development environment with every dependencies just run the following command:

```bash
$ make up
```

The inverse of that command is of course:

```bash
$ make down
```

which removes everything.


### Configuration

Create a `config/config.toml` based on `config/config.toml.example`:

```bash
$ make config/config.toml
```

**Note:** If you followed the quick start guide this file should already exist.
 
As of now the example config enables OAuth2 based authentication. It can be changed by modifying the example.

OAuth2 based authentication requires GitHub application, this can be created by following this 
[tutorial](https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/).
Please set the `clientid` and the `clientsecret` in the auth section, with the GitHub generated values.

> If you are not using HTTPS set auth.secureCookie = false, otherwise you won't be able to login via HTTP.


### Environment

The development environment uses Docker Compose to create an isolated area for Pipeline.

You can easily start it by executing: 

```bash
$ make start
``` 

This will create a `mysql`, `adminer` and `vault` container:
 - Adminer MySQL GUI: http://localhost:8080 login with username/password `sparky/sparky123`
 - Vault GUI: http://localhost:8200 login with token found in `cat ~/.vault-token`

**Note:** If you want to customize mount points and port mappings, create a `docker-compose.override.yml` file via
`make docker-compose.override.yml` and edit the file manually. Please note that you might need to edit the application
configuration file as well.


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

#### User and organization whitelist

If you enable user and organization whitelist with:

```bash
export PIPELINE_AUTH_WHITELISTENABLED=true
```

the Pipeline will limit which users can register, this list is stored in the `whitelisted_auth_identities` table, you can add users or organizations to this table (if you add an organization all members of the organization are allowed to register):

- Add `banzaicloud` organization for example:

    Get the `banzaicloud` organization information from: https://api.github.com/orgs/banzaicloud

    ```sql
    INSERT INTO whitelisted_auth_identities (created_at, updated_at, provider, type, login, uid) VALUES (NOW(), NOW(), "github", "Organization", "banzaicloud", 32848483)
    ```

- Add `bonifaido` user for example:

    Get the `bonifaido` user information from: https://api.github.com/users/bonifaido

    ```sql
    INSERT INTO whitelisted_auth_identities (created_at, updated_at, provider, type, login, uid) VALUES (NOW(), NOW(), "github", "User", "bonifaido", 23779)
    ```

#### Anchore Engine

If you need to access local anchore server, you'll have to start development environment with `anchorestart` instead of `start`

```bash
$ make anchorestart
```
