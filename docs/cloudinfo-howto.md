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

`GET orgs/{orgId}/cloudinfo/amazon` returns the cloud type and regexp for name:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$"
}
```

#### Supported locations

`GET orgs/{{orgId}}/cloudinfo/amazon?fields=location&secret_id={{secret_id}}`:

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
`GET orgs/{{orgId}}/cloudinfo/amazon?fields=image&tags=0.3.0&location=eu-west-1&secret_id={{secret_id}}`:

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
`GET orgs/{{orgId}}/cloudinfo/amazon?fields=instanceType&location=eu-west-1&secret_id={{secret_id}}`

Response:
```
{
    "type": "amazon",
    "nameRegexp": "^[A-z0-9-_]{1,255}$",
    "instanceType": {
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
`GET orgs/{{orgId}}/cloudinfo/amazon?fields=location&fields=image&fields=instanceType&tags=0.3.0&location=eu-west-1&secret_id={{secret_id}}`:

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
    },
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

## Azure
`GET orgs/{orgId}/cloudinfo/azure` returns the cloud type and regexp for name:

#### Supported locations
`GET orgs/{{orgId}}/cloudinfo/azure?fields=location&secret_id={{secret_id}}`:

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
`GET orgs/{{orgId}}/cloudinfo/azure?fields=instanceType&location=eastus&secret_id={{secret_id}}`:

Response:
```
{
    "type": "azure",
    "nameRegexp": "^[a-z0-9_]{0,31}[a-z0-9]$",
    "instanceType": {
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
`GET orgs/{orgId}/cloudinfo/azure?fields=k8sVersion&location=eastus&secret_id={{secret_id}}`:

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
`GET orgs/{orgId}/cloudinfo/azure?fields=location&fields=instanceType&fields=k8sVersion&location=eastus&secret_id={{secret_id}}`:

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
    "instanceType": {
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
`GET orgs/{orgId}/cloudinfo/google` returns the cloud type and regexp for name:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$"
}
```

#### Supported locations
`GET orgs/{orgId}/cloudinfo/google?fields=location&secret_id={{secret_id}}`:

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
`GET orgs/{orgId}/cloudinfo/google?fields=instanceType&location=asia-east1-a&secret_id={{secret_id}}`:

Response:
```
{
    "type": "google",
    "nameRegexp": "^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$",
    "instanceType": {
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
`GET orgs/{orgId}/cloudinfo/google?fields=k8sVersion&location=us-central1-a&secret_id={{secret_id}}`:

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
`GET orgs/{orgId}/cloudinfo/google?fields=location&fields=instanceType&fields=k8sVersion&location=us-central1-a&secret_id={{secret_id}}`:

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
    "instanceType": {
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
