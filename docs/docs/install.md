#### Cloud provider support

Pipeline proudly uses (and contributes) to [Kubicorn](http://kubicorn.io) - although AWS, Azure, Google and Digital Ocean is among supported providers, for the time being Pipeline supports AWS only. Other providers are coming soon.

#### AWS Authentication

There are three ways to authenticate against AWS.

 * Environment Credentials - export the two environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` so that `Pipeline` can pick it up as described in the next steps:
 
    ```
    $ export AWS_ACCESS_KEY_ID=***************
    $ export AWS_SECRET_ACCESS_KEY=*****************************************
    ```

 * Shared Credentials file - The `~/.aws/credentials` file stores your credentials based on a profile name 

 * EC2 Instance Role Credentials - Use EC2 Instance Role to assign credentials to application running on an EC2 instance. 

#### SSH key setup

You will need a `phasspraseless` SSH key, names `id_rsa`. To generate SSH keys on Mac OS X, follow these steps: `ssh-keygen -t rsa`

The ssh-keygen utility prompts you to indicate where to store the key. Press ENTER key to accept the default location. The ssh-keygen utility prompts for a passphrase. Hit ENTER key to accept the default **(no passphrase)**.

Your private key has been saved in `/Users/myname/.ssh/id_rsa`.
Your public key has been saved in `/Users/myname/.ssh/id_rsa.pub`.

### Installation

You have three options to try out Pipeline. 

#### Cloudformation

The easiest is by running a Pipeline control plane using the following Cloudformation [template](https://github.com/banzaicloud/control-plane-k8s-cf/blob/master/control-plane.template). 

#### The DIY way

* Have [Go](https://golang.org/doc/install) installed and configured - 1.8.3+
* Install Go's dependency management tool, [dep](http://github.com/golang/dep)
*  ~~Install dependencies by running dep ensure~~
* Clone [Pipeline](https://github.com/banzaicloud/pipeline), checkout the `master` branch and run `make build`. That will create the `pipeline executable binary

_Note: Due to the current dependency hell introduced by a few K8S packages (apimachinery, go-client, helm), please don't issue `dep ensure -update` or `dep prune` until these are not fixed upstream. Pipeline dependencies are managed `semi-manually` (containerized) and triggered by CircleCI.

Clusters are created in different cloud regions using different images. Currently this is the list of AWS images available/publish in the following regions:

```
eu-central-1: ami-a208bccd
eu-west-1: ami-c46caabd
eu-west-2: ami-e1405385
us-east-1: ami-d67e60ad
us-east-2: ami-f4260491
us-west-1: ami-53e4d333
us-west-2: ami-0904f271
```

#### Running in Docker compose

Export your environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`:

```
$ export AWS_ACCESS_KEY_ID=***************
$ export AWS_SECRET_ACCESS_KEY=*****************************************
```

run `docker-compose up`
