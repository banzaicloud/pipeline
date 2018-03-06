Starting from the [0.2.0](https://github.com/banzaicloud/pipeline/tree/0.2.0) version Pipeline supports managed Kubernetes clusters on Azure [AKS](https://docs.microsoft.com/en-us/azure/aks/) as well.

For simplicity the instruction steps are presented through an example specifically how to hook a Spark application into a CI/CD workflow to run it on managed Kubernetes AKS/Azure.

### Getting Started

It's assumed that the source of the Spark application is stored in [GitHub](http://github.com).

The [Pipeline Control Plane](https://github.com/banzaicloud/pipeline-cp-launcher/tree/0.2.0) takes care of creating a Kubernetes cluster on the desired cloud provider and executing the steps of the CI/CD flow can be hosted on both `AWS` and `Azure`. See details below for how to launch `Pipeline Control Plane` on `AWS` and `Azure`.

### General prerequisites

1. Account on [GitHub](http://github.com)
1. Repository on [GitHub](http://github.com) for the Spark application source code

### Hosting the control plane on AWS 

Hosting `Pipeline Control Plane` and creating Kubernetes clusters on **`AWS`**

   1. [AWS](https://portal.aws.amazon.com/billing/signup?type=enterprise#/start) account
   1. AWS [EC2 key pair](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html)
   
### Hosting the control plane on Azure

Hosting `Pipeline Control Plane` and creating Kubernetes clusters on **`Azure`**

   1. [Azure](https://portal.azure.com) subscription with AKS service enabled.
   1. Obtain a Client Id, Client Secret and Tenant Id for a Microsoft Azure Active Directory. These information can be retrieved from the portal, but the easiest and fastest way is to use the Azure CLI tool.<br>  
   
```bash
$ curl -L https://aka.ms/InstallAzureCli | bash
$ exec -l $SHELL
$ az login
```

You should get something like:

```json
{

  "appId": "1234567-1234-1234-1234-1234567890ab",
  "displayName": "azure-cli-2017-08-18-19-25-59",
  "name": "http://azure-cli-2017-08-18-19-25-59",
  "password": "1234567-1234-1234-be18-1234567890ab",
  "tenant": "7654321-1234-1234-ee18-9876543210ab"
}
```

* `appId` is the Azure Client Id
* `password` is the Azure Client Secret
* `tenant` is the Azure Tenant Id

In order to get Azure Subscription Id run:

```sh
az account show --query id
```

### Register the OAuth application on GitHub

Setup your Pipeline GitHub OAuth application according to: [this guilde](./github-app.md)

### Launch Pipeline Control Plane on `AWS`

The easiest way for running a Pipeline Control Plane is to use a [Cloudformation](https://aws.amazon.com/cloudformation/) template.

* Navigate to: https://eu-west-1.console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/new

* Select `Specify an Amazon S3 template URL` and add the URL to our template `https://s3-eu-west-1.amazonaws.com/cf-templates-grr4ysncvcdl-eu-west-1/2018026em9-new.templatee93ate9mob7`

  <a href="images/howto/ControlPlaneFromTemplate.png" target="_blank"><img src="images/howto/ControlPlaneFromTemplate.png" height="230"></a>

* Fill in the following fields on the form:

  * **Stack name**
    * specify a name for the Control Plane deployment

      <a href="images/howto/StackName.png"><img src="images/howto/StackName.png" height="130"></a>

  * **AWS Credentials**
     * Amazon access key id - specify your [access key id](http://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys)
     * Amazon secret access key - specify your [secret access key](http://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys)

     <a href="images/howto/AwsCred.png"><img src="images/howto/AwsCred.png" height="150"></a>

  * **Azure Credentials and Information** - _needed only for creating Kubernetes clusters on `Azure`_
     * AzureClientId - see how to get Azure Client Id above
     * AzureClientSecret - see how to get Azure Client Secret above
     * AzureSubscriptionId - your Azure Subscription Id
     * AzureTenantId - see how to get Azure Client Tenant Id above

     <a href="images/howto/AzureCred.png"><img src="images/howto/AzureCred.png" height="200"></a>

  * **Control Plane Instance Config**
     * InstanceName - name of the EC2 instance that will host the Control Plane
     * ImageId - pick the image id from the  [README](https://github.com/banzaicloud/pipeline-cp-launcher/blob/0.2.0/README.md)
     * KeyName - specify your AWS [EC2 key pair](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html)

     <a href="images/howto/ControlPlaneInstanceConfig.png"><img src="images/howto/ControlPlaneInstanceConfig.png" height="180"></a>

  * **Banzai-Ci Credentials**
     * Orgs - comma-separated list of Github organizations whose members to grant access to use Banzai Cloud Pipeline's CI/CD workflow
     * Github Client - GitHub OAuth `Client Id`
     * Github Secret - Github OAuth `Client Secret`

      <a href="images/howto/CloudFormulationDetails3.png"><img src="images/howto/CloudFormulationDetails3.png" height="150"></a>

  * **Grafana Dashboard**
     * Grafana Dashboard Password - specify password for accessing Grafana dashboard with defaults specific to the application

     <a href="images/howto/GrafanaCred.png"><img src="images/howto/GrafanaCred.png" height="100"></a>

  * **Prometheus Dashboard**
     * Prometheus Password - specify password for accessing Prometheus that collects cluster metrics

      <a href="images/howto/PrometheusCred.png"><img src="images/howto/PrometheusCred.png" height="100"></a>

  * **Advanced Pipeline Options**
     * PipelineImageTag - specify `0.2.0` for using current stable Pipeline release.

     <a href="images/howto/AdvencedPipOpt.png"><img src="images/howto/AdvencedPipOpt.png" height="150"></a>

  * **Slack Credentials**
       * this section is optional. Complete this section to receive  cluster related alerts through a [Slack](https://slack.com) push notification channel.

  * **Alert SMTP Credentials**
     * this section is optional. Fill this section to receive cluster related alerts through email.

* Finish the wizard to create a `Control Plane` instance.
* Take note of the PublicIP of the created Stack. We refer to this as the PublicIP of `Control Plane`.

    <a href="images/howto/CloudFormulationDetails5.png"><img src="images/howto/CloudFormulationDetails5.png" height="250"></a>

* Go back to the earlier created GitHub OAuth application and modify it. Set the `Authorization callback URL` field to `http://{control_plane_public_ip}/auth/github/callback`

### Launch Pipeline Control Plane on `Azure`

The easiest way for running a Pipeline Control Plane is deploying it using an [ARM](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview) template.

* Navigate to: https://portal.azure.com/#create/Microsoft.Template

* Click `Build your own template in editor` and copy-paste the content of [ARM template](https://raw.githubusercontent.com/banzaicloud/pipeline-cp-launcher/0.2.0/control-plane-arm.json) into the editor then click `Save`

  <a href="images/howto/ARMCreate.png" target="_blank"><img src="images/howto/ARMCreate.png" height="130"></a><br>
  <a href="images/howto/ARMEditor.png" target="_blank"><img src="images/howto/ARMEditor.png" height="230"></a>

  * **Resource group** - We recommend creating a new `Resource Group` for the deployment as later will be easier to clean up all the resources created by the deployment

    <a href="images/howto/ARMRGroup.png" target="_blank"><img src="images/howto/ARMRGroup.png" height="180"></a>

  * **Specify SSH Public Key**

    <a href="images/howto/ARMPubKey.png" target="_blank"><img src="images/howto/ARMPubKey.png" height="110"></a>

  * **SMTP Server Address/User/Password/From**
    * these are optional. Fill this section to receive cluster related alerts through email.

  * **Slack Webhook Url/Channel**
    * this section is optional. Complete this section to receive  cluster related alerts through a [Slack](https://slack.com) push notification channel.

  * **Prometheus Dashboard**
    * Prometheus Password - specify password for accessing Prometheus that collects cluster metrics

    <a href="images/howto/ARMPrometheusCred.png"><img src="images/howto/ARMPrometheusCred.png" height="70"></a>

  * **Grafana Dashboard**
     * Grafana Dashboard Password - specify password for accessing Grafana dashboard with defaults specific to the application

     <a href="images/howto/ARMGrafanaCred.png"><img src="images/howto/ARMGrafanaCred.png" height="70"></a>
  
  * **Banzai-Ci Credentials**
     * Orgs - comma-separated list of Github organizations whose members to grant access to use Banzai Cloud Pipeline's CI/CD workflow
     * Github Client - GitHub OAuth `Client Id`
     * Github Secret - Github OAuth `Client Secret`

      <a href="images/howto/ARMCICD.png"><img src="images/howto/ARMCICD.png" height="90"></a>

  * **Azure Credentials and Information**
     * Azure Client Id - see how to get Azure Client Id above
     * Azure Client Secret - see how to get Azure Client Secret above
     * Azure Subscription Id - your Azure Subscription Id
     * Azure Tenant Id - see how to get Azure Tenant Id above

     <a href="images/howto/images/howto/ARMAzureCreds.png"><img src="images/howto/ARMAzureCreds.png" height="130"></a>

  * Finish the wizard to create a `Control Plane` instance.
  * Open the `Resource Group` that was specified for the deployment

      <a href="images/howto/CPRGroup.png"><img src="images/howto/CPRGroup.png" height="90"></a>

  * Take note of the PublicIP of the deployed `Control Plane`.

    <a href="images/howto/AzureCPPubIP.png"><img src="images/howto/AzureCPPubIP.png" height="200"></a>

### Define `.pipeline.yml` pipeline workflow configuration for your Spark application

The steps of the workflow executed by the CI/CD flow are described in the  `.pipeline.yml` file that must be placed under the root directory of the source code of the Spark application. The file has to be pushed into the GitHub repo along with the source files of the application.

There is an example Spark application [spark-pi-example](https://github.com/banzaicloud/spark-pi-example/tree/0.2.0) that can be used for trying out the CI/CD pipeline.

>Note: Fork this repository into your own repository for this purpose!).

For setting up your own spark application for the workflow you can start from the `.pipeline.yml` configuration file from `spark-pi-example` and customize it.

The following sections needs to be modified:

- the command for building your application

  ```yaml
  remote_build:
    ...
    original_commands:
      - mvn clean package -s settings.xml
  ```

- the Main class of your application

  ```yaml
  run:
    ...
    spark_class: banzaicloud.SparkPi
  ```

- the name of your application

  ```yaml
  run:
    ...
    spark_app_name: sparkpi
  ```

- the application artifact

  This is the relative path to the `jar` of your Spark application. This is the `jar` generated by the [build command](#command-for-building-your-application)

  ```yaml
  run:
    ...
    spark_app_source: target/spark-pi-1.0-SNAPSHOT.jar
  ```

-  the application arguments

  ```yaml
  run:
    ...
    spark_app_args: 1000
  ```

### Grant access to desired GitHub organizations

Navigate to `http://{control_plane_public_ip}` in your web browser and grant access for the organizations that contain the GitHub repositories that you want to hook into the CI/CD workflow. Then click authorize access.

All the services of the Pipeline may take some time to fully initialize, thus the page may not load at first. Please give it some time and retry.

### Hook repositories to CI/CD flow

Navigate to `http://{control_plane_public_ip}`  - this will bring you to the CI/CD user interface. Select `Repositories` from top left menu. This will list all the repositories that the Pipeline has access to.
Select repositories desired to be hooked to the CI/CD flow.

<a href="images/howto/EnableRepoCI.png" target="_blank"><img src="images/howto/EnableRepoCI.png"></a>

### CI/CD secrets

For the hooked repositories set the following secrets :

<a href="images/howto/RepoSecretCI.png" target="_blank"><img src="images/howto/RepoSecretCI.png"></a>
<a href="images/howto/RepoSecretMenuCI.png" target="_blank"><img src="images/howto/RepoSecretMenuCI.png"></a>

* `plugin_endpoint` - specify `http://{control_plane_public_ip}/pipeline/api/v1`

  <a href="images/howto/RepoSecretPluginEndPointCI.png" target="_blank"><img src="images/howto/RepoSecretPluginEndPointCI.png"></a>

* `plugin_username` - specify the same user name as for **Banzai Pipeline Credentials**

  <a href="images/howto/RepoSecretPluginUserNameCI.png" target="_blank"><img  src="images/howto/RepoSecretPluginUserNameCI.png"></a>

* `plugin_password` - specify the same password as for **Banzai Pipeline Credentials**

  <a href="images/howto/RepoSecretPluginPasswordCI.png" target="_blank"><img src="images/howto/RepoSecretPluginPasswordCI.png"></a>

### Submit your changes

Modify the source code of your Spark application, commit the changes and push it to the repository on GitHub. The Pipeline gets notified through GitHub webhooks about the commits and will trigger the flow described in the `.pipeline.yml` file of the watched repositories.

### Monitor running workflows

The running CI/CD jobs can be monitored and managed at `http://{control_plane_public_ip}/account/repos`


<a href="images/howto/BuildMenuCI.png" target="_blank"><img src="images/howto/BuildMenuCI.png"></a>

<a href="images/howto/JobCI.png" target="_blank"><img src="images/howto/JobCI.png"></a>

In order to check the logs of the CI/CD workflow steps, click on the desired commit message on the UI.

<a href="images/howto/JobCIBuild.png"><img src="images/howto/JobCIBuild.png"></a>
<br>
<a href="images/howto/SparkPiSuccess.png"><img src="images/howto/SparkPiSuccess.png" height="370"></a>

Once configured the Spark application will be built, deployed and executed for every commit pushed to the project's repository. The progress of the workflow can be followed by clicking on the small orange dot beside the commit on the GitHub UI.

Our git repos with example projects that contain pipeline workflow configurations:

- [Spark PDI Example](https://github.com/banzaicloud/spark-pdi-example/tree/0.2.0)
- [Zeppelin PDI Example](https://github.com/banzaicloud/zeppelin-pdi-example/tree/0.2.0)
- [Spark Pi Example](https://github.com/banzaicloud/spark-pi-example/tree/0.2.0)
