### Cloud info howto

## Common

`GET /orgs/{orgId}/cloudinfo` returns the supported cloud types and keys.

Response:
 ```
 {
    "items": [
        {
            "name": "Amazon Web Services",
            "key": "amazon"
        },
        {
            "name": "Azure Container Service",
            "key": "azure"
        },
        {
            "name": "Google Kubernetes Engine",
            "key": "google"
        },
        {
            "name": "Kubernetes Cluster",
            "key": "kubernetes"
        }
    ]
}
 ```

 `GET orgs/{orgId}/cloudinfo/filters` returns the supported filter keys. Response:

 ```
 {
    "keys": [
        "location",
        "instanceType",
        "k8sVersion"
    ]
}
```

## Amazon

`POST orgs/{orgId}/cloudinfo/amazon` with empty body returns the cloud type and regexp for name:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$"
}
```

#### Supported locations

`POST orgs/{orgId}/cloudinfo/amazon` with body:
```
{
	"filter": {
		"fields": [ "location" ]
	}
}
```
Response:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$",
    "locations": [
        "ap-south-1",
        "eu-west-3",
        "eu-west-2",
        "eu-west-1",
        "ap-northeast-2",
        "ap-northeast-1",
        "sa-east-1"
    ]
}
```

#### Supported images
`POST orgs/{orgId}/cloudinfo/amazon` with body:
```
{
	"filter": {
		"fields": [ "image" ],
		"image": {
			"tags": [ "0.3.0" ],
			"location": "eu-west-1"
		}
	}
}
```

Response:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$",
    "image": {
        "eu-west-1": [
            "ami-6202561b",
            "ami-ece5b095"
        ]
    }
}
```

#### Supported instanceTypes
`POST orgs/{orgId}/cloudinfo/amazon` with body:
```
{
	"filter": {
		"fields": [ "instanceType" ],
		"instanceType": {
			"location": "eu-west-1"
		}
	}
}
```

Response:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$",
    "nodeInstanceType": {
        "eu-west-1": [
            "t2.nano",
            "t2.micro",
            "t2.small",
            "t2.medium",
            "t2.large",
            "t2.xlarge",
            "t2.2xlarge",
            "m5.large"
        ]
    }
}
```


#### All supported fields
`POST orgs/{orgId}/cloudinfo/amazon` with body:
```
{
	"filter": {
		"fields": [
			"location",
			"image"
		],
		"image": {
			"tags": [ "0.3.0" ],
			"location": "eu-west-1"
		}
	}
}
```
Response:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$",
    "locations": [
        "ap-south-1",
        "eu-west-3",
        "eu-west-2",
        "eu-west-1",
        "ap-northeast-2",
        "ap-northeast-1",
        "sa-east-1"
    ],
    "image": {
        "eu-west-1": [
            "ami-6202561b",
            "ami-ece5b095"
        ]
    }
}
```

## Azure
`POST orgs/{orgId}/cloudinfo/azure` with empty body returns the cloud type and regexp for name:
```
{
    "type": "azure",
    "nameRegexp": "^[a-z0-9_]{0,31}[a-z0-9]$"
}
```

#### Supported locations
`POST orgs/{orgId}/cloudinfo/azure` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [ "location" ]
	}
}
```
Response:
```
{
    "type": "azure",
    "nameRegexp": "^[a-z0-9_]{0,31}[a-z0-9]$",
    "locations": [
        "eastasia",
        "southeastasia",
        "centralus",
        "eastus",
        "eastus2",
        "westus"
    ]
}
```
#### Supported node instance types
`POST orgs/{orgId}/cloudinfo/azure` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [ "instanceType" ],
		"instanceType":{
			"location": "eastus"
		}
	}
}
```
Response:
```
{
    "type": "azure",
    "nameRegexp": "^[a-z0-9_]{0,31}[a-z0-9]$",
    "nodeInstanceType": {
        "eastus": [
            "Standard_B1ms",
            "Standard_B1s",
            "Standard_B2ms",
            "Standard_B2s",
            "Standard_B4ms",
            "Standard_B8ms"
        ]
    }
}
```
#### Supported Kubernetes versions
`POST orgs/{orgId}/cloudinfo/azure` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [ "k8sVersion" ],
		"k8sVersion":{
			"location": "eastus"
		}
	}
}
```
Response:
```
{
    "type": "azure",
    "nameRegexp": "^[a-z0-9_]{0,31}[a-z0-9]$",
    "kubernetes_versions": [
        "1.7.7",
        "1.7.9",
        "1.6.11",
        "1.8.6"
    ]
}
```

#### All supported fields
`POST orgs/{orgId}/cloudinfo/amazon` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [
			"location",
			"instanceType",
			"k8sVersion"
		],
		"instanceType":{
			"location": "eastus"
		},
		"k8sVersion": {
			"location": "eastus"
		}
	}
}
```
Response:
```
{
    "type": "azure",
    "nameRegexp": "^[a-z0-9_]{0,31}[a-z0-9]$",
    "locations": [
        "eastasia",
        "southeastasia",
        "centralus",
        "eastus",
        "eastus2",
        "westus"
    ],
    "nodeInstanceType": {
        "eastus": [
            "Standard_B1ms",
            "Standard_B1s",
            "Standard_B2ms",
            "Standard_B2s",
            "Standard_B4ms",
            "Standard_B8ms"
        ]
    },
    "kubernetes_versions": [
        "1.8.2",
        "1.6.9",
        "1.7.9",
        "1.8.7",
        "1.7.7",
        "1.6.11",
        "1.8.6"
    ]
}
```

## Google
`POST orgs/{orgId}/cloudinfo/google` with empty body returns the cloud type and regexp for name:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$"
}
```

#### Supported locations
`POST orgs/{orgId}/cloudinfo/google` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [ "location" ]
	}
}
```
Response:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$",
    "locations": [
        "us-east1-b",
        "us-east1-c",
        "us-east1-d",
        "us-east4-c",
        "us-east4-b",
        "us-east4-a"
    ]
}
```
#### Supported node instance types
`POST orgs/{orgId}/cloudinfo/google` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [ "instanceType" ],
		"instanceType":{
			"location": "asia-east1-a"
		}
	}
}
```
Response:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$",
    "nodeInstanceType": {
        "asia-east1-a": [
            "f1-micro",
            "g1-small",
            "n1-highcpu-16",
            "n1-highcpu-2"
        ]
    }
}
```
#### Supported Kubernetes versions
`POST orgs/{orgId}/cloudinfo/google` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [ "k8sVersion" ],
		"k8sVersion": {
			"location": "us-central1-a"
		}
	}
}
```
Response:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$",
    "kubernetes_versions": {
        "defaultClusterVersion": "1.8.8-gke.0",
        "defaultImageType": "COS",
        "validImageTypes": [
            "UBUNTU",
            "COS"
        ],
        "validMasterVersions": [
            "1.9.6-gke.1",
            "1.9.6-gke.0",
            "1.9.4-gke.1",
            "1.9.3-gke.0"
        ],
        "validNodeVersions": [
            "1.9.6-gke.1",
            "1.9.6-gke.0",
            "1.9.4-gke.1",
            "1.9.3-gke.0"
        ]
    }
}
```
#### All supported fields
`POST orgs/{orgId}/cloudinfo/google` with body:
```
{
	"secret_id": "{{secret_id}}",
	"filter": {
		"fields": [
			"location",
			"instanceType",
			"k8sVersion"
		],
		"instanceType":{
			"location": "asia-east1-a"
		},
		"k8sVersion": {
			"location": "us-central1-a"
		}
	}
}
```
Response:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$",
    "locations": [
        "us-east1-b",
        "us-east1-c",
        "us-east1-d",
        "us-east4-c",
        "us-east4-b",
        "us-east4-a"
    ],
    "nodeInstanceType": {
        "asia-east1-a": [
            "f1-micro",
            "g1-small",
            "n1-highcpu-16",
            "n1-highcpu-2"
        ]
    },
    "kubernetes_versions": {
        "defaultClusterVersion": "1.8.8-gke.0",
        "defaultImageType": "COS",
        "validImageTypes": [
            "COS",
            "UBUNTU"
        ],
        "validMasterVersions": [
            "1.9.6-gke.1",
            "1.9.6-gke.0",
            "1.9.4-gke.1",
            "1.9.3-gke.0"
        ],
        "validNodeVersions": [
            "1.9.6-gke.1",
            "1.9.6-gke.0",
            "1.9.4-gke.1",
            "1.9.3-gke.0"
        ]
    }
}
```
