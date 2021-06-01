# Developer Guide

## How to run Pipeline in your local dev environment

### Prerequisites

- Make
- Docker (with Compose)
- Account on Github (optional)
- Account on Google (optional)

### Authentication setup

At least one of the followings has to be configured:

- Setup your Pipeline GitHub OAuth application according to [this guide](auth/github.md)
- Setup your Pipeline GitLab OAuth application according to [this guide](auth/gitlab.md)
- Setup your Pipeline Google OAuth application according to [this guide](auth/google.md)
- Setup your Pipeline with LDAP authentication according to [this guide](auth/ldap.md)
- Use static Email/Password authentication following the example in `etc/config/dex.yml` (staticPasswords sections)

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

Create a `config/config.yaml` and `etc/config/dex.yml` config file based on their `config/*.dist` counterparts with:

```bash
$ make config/config.yaml etc/config/dex.yml
```

**Note:** If you followed the quick start guide this file should already exist.

As of now the example config enables OAuth2 based authentication. It can be changed by modifying the example.

OAuth2 based authentication requires a GitHub/Google OAuth2 application, this can be created by following this
[GitHub](auth/github.md), [GitLab](auth/gitlab.md) or the [Google](auth/google.md) tutorial.
Please set the `clientId` and the `clientSecret` in `dex.yml`'s `connectors:` section.

> If you are not using HTTPS set auth.cookie.secure = false, otherwise you won't be able to login via HTTP and you might be getting 401 errors, locally you should set it to `false`.

### Environment

The development environment uses Docker Compose to create an isolated area for Pipeline.

You can easily start it by executing:

```bash
$ make start
```

This will create a `mysql` and `vault` container:
 - Vault GUI: http://localhost:8200 login with token found in `cat ~/.vault-token`

**Note:** If you want to customize mount points and port mappings, create a `docker-compose.override.yml` file via
`make docker-compose.override.yml` and edit the file manually. Please note that you might need to edit the application
configuration file as well.


#### Set Required Environment Variables

For accessing Vault the `VAULT_ADDR` env var has to be set, Pipeline stores JWT access tokens there.

```bash
export VAULT_ADDR=http://127.0.0.1:8200
```

#### Start Pipeline

Once you have the docker containers running for the development environment, you should be able to start the pipeline platform.

You can install and then run it with:
```bash
$ make build
$ build/pipeline
```

You will also need to run a Worker for background jobs:
```bash
$ build/worker
```

(Optionally, you could also build and run with VSCode or Goland.)

If you happen to get an error similar to this on the first run:
```
Error 1146: Table 'pipeline.amazon_eks_profiles' doesn't exist
```

You should set `autoMigrate = true` in the database section in the `config/config.yaml` file.

You should now be able to log in on the Pipeline UI: http://localhost:4200/ui

#### Acquiring an access token

For accessing the Pipeline one has to be authenticated and registered via Dex first.

For programmatic API access an access token has to be generated.

Tokens can be generated only with a browser (for now), to do that please use the following URL to login first:

- For local usage:
    ```bash
    https://localhost:9090/auth/dex/login
    ```

- For on-cloud usage:
    ```bash
    https://{control_plane_public_ip}/auth/dex/login
    ```

Please authenticate yourself with Dex. If everything is done correctly you will be redirected.
The browser session already contains the generated token in a cookie. An API token can be generated via:

- For local usage:
    ```bash
    https://localhost:9090/pipeline/api/v1/tokens
    ```

- For on-cloud usage:
    ```bash
    https://{control_plane_public_ip}/pipeline/api/v1/tokens
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
go get -u -v sigs.k8s.io/aws-iam-authenticator/cmd/aws-iam-authenticator
```

#### EKS versions and images

[How to update default EKS images](eks-images.md)

#### Anchore Engine

If you need to access local anchore server, uncomment the related services in `docker-compose.override.yml`
and restart the environment with `make start`.
You can specify your own default Anchore policy bundle by adding a `json` file to the `config/anchore/policies` directory.

#### Accessing Pipeline API from the cluster

If you want to launch PKE clusters, you will need to ensure that the pke-tool running on the cluster will access the Pipeline API.
In a development environment you can do this for example with the following
[ngrok](https://ngrok.com/) command: `ngrok http https://localhost:9090` (HTTPS
requires free registration).

You will also need to adjust the `pipeline.external.url` configuration value.
In the `pipeline` section of `config/config.yaml` you can add the value like below:

```yaml
pipeline:
    external:
        # Base URL where the end users can reach this pipeline instance
        url: "http://<YOUR_NGROK_NUMBER>.ngrok.io/pipeline"
```

#### Helm S3 repositories

To be able to handle S3 repositories with Pipeline helm-s3 need to be installed in case you have helm installed on your machine:

```bash
helm plugin install https://github.com/banzaicloud/helm-s3.git
```

In case you don't have helm installed you can download helm-s3 plugin from [here](https://github.com/banzaicloud/helm-s3/releases), extract it to a `helm-plugins` folder and set `HELM_PLUGINS=helm-plugins` when starting Pipeline.
