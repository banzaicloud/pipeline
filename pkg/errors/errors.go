package errors

import "errors"

// ### [ Errors ] ### //
var (
	ErrorNotSupportedCloudType          = errors.New("Not supported cloud type")
	ErrorAmazonClusterNameRegexp        = errors.New("Up to 255 letters (uppercase and lowercase), numbers, hyphens, and underscores are allowed.")
	ErrorAmazonFieldIsEmpty             = errors.New("Required field 'amazon' is empty.")
	ErrorAmazonMasterFieldIsEmpty       = errors.New("Required field 'master' is empty.")
	ErrorAmazonImageFieldIsEmpty        = errors.New("Required field 'image' is empty ")
	ErrorAmazonNodePoolFieldIsEmpty     = errors.New("At least one 'nodePool' is required.")
	ErrorAmazonInstancetypeFieldIsEmpty = errors.New("Required field 'instanceType' is empty ")
	ErrorNodePoolMinMaxFieldError       = errors.New("'maxCount' must be greater than 'minCount'")
	ErrorMinFieldRequiredError          = errors.New("'minCount' must be set in case 'autoscaling' is set to true")
	ErrorMaxFieldRequiredError          = errors.New("'maxCount' must be set in case 'autoscaling' is set to true")
	ErrorGoogleClusterNameRegexp        = errors.New("Name must start with a lowercase letter followed by up to 40 lowercase letters, numbers, or hyphens, and cannot end with a hyphen.")
	ErrorAzureClusterNameRegexp         = errors.New("Only numbers, lowercase letters and underscores are allowed under name property. In addition, the value cannot end with an underscore, and must also be less than 32 characters long.")
	ErrorAzureClusterNameEmpty          = errors.New("The name should not be empty.")
	ErrorAzureClusterNameTooLong        = errors.New("Cluster name is greater than or equal 32")
	ErrorAzureCLusterStageFailed        = errors.New("cluster stage is 'Failed'")
	ErrorNotDifferentInterfaces         = errors.New("There is no change in data")
	ErrorReconcile                      = errors.New("Error during reconcile")
	ErrorEmptyUpdateRequest             = errors.New("Empty update cluster request")
	ErrorClusterNotReady                = errors.New("Cluster not ready yet")
	ErrorNilCluster                     = errors.New("<nil> cluster")
	ErrorWrongKubernetesVersion         = errors.New("Wrong kubernetes version for master/nodes. The required minimum kubernetes version is 1.8.x ")
	ErrorDifferentKubernetesVersion     = errors.New("Different kubernetes version for master and nodes")
	ErrorLocationEmpty                  = errors.New("Location field is empty")
	ErrorNodeInstanceTypeEmpty          = errors.New("instanceType field is empty")
	ErrorRequiredLocation               = errors.New("location is required")
	ErrorRequiredSecretId               = errors.New("Secret id is required")
	ErrorCloudInfoK8SNotSupported       = errors.New("Not supported key in case of amazon")
	ErrorNodePoolNotProvided            = errors.New("At least one 'nodepool' is required for creating or updating a cluster")
	ErrorOnlyOneNodeModify              = errors.New("only one node can be modified at a time")
	ErrorNotValidLocation               = errors.New("not valid location")
	ErrorNotValidMasterImage            = errors.New("not valid master image")
	ErrorNotValidNodeImage              = errors.New("not valid node image")
	ErrorNotValidNodeInstanceType       = errors.New("not valid nodeInstanceType")
	ErrorNotValidMasterVersion          = errors.New("not valid master version")
	ErrorNotValidNodeVersion            = errors.New("not valid node version")
	ErrorNotValidKubernetesVersion      = errors.New("not valid kubernetesVersion")
	ErrorResourceGroupRequired          = errors.New("resource group is required")
	ErrorProjectRequired                = errors.New("project is required")
	ErrorNodePoolNotFoundByName         = errors.New("nodepool not found by name")
	ErrorNoInfrastructureRG             = errors.New("no infrastructure resource group found")
	ErrStateStorePathEmpty              = errors.New("statestore path cannot be empty")
)
