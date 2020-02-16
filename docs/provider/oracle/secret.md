# How to create an Oracle secret

To create an Oracle secret you need these fields:
- [`region`](#region)
- [`tenancy_ocid`](#tenancy_ocid)
- [`user_ocid`](#user_ocid)
- [`compartment_ocid`](#compartment_ocid)
- [`api_key`](#api_key)
- [`api_key_fingerprint`](#api_key_fingerprint)


## Where these fields found

#### region

The recommended region is `eu-frankfurt-1`.

#### user_ocid

Navigate to `User settings` in the user menu. Under `User information` find the `user_ocid`.

<p align="center">
<img src="images/oracle_user_ocid_1.png" width="700">
</p>

<p align="center">
<img src="images/oracle_user_ocid_2.png" width="700">
</p>

#### tenancy_ocid

Tenancy ocid can be found on the `Tenancy: ...` page of the user menu (like the user ocid on the User settings page).

#### compartment_ocid

From the menu choose `Identitiy` menu item. Under `Compartments` found a list of your Compartments. If the list is empty, create a new Compartment.

<p align="center">
<img src="images/oracle_compartments_1.png" width="700">
</p>

<p align="center">
<img src="images/oracle_compartments_2.png" width="700">
</p>

#### api_key

> If you haven't already, create a .oci directory to store the credentials: ```mkdir ~/.oci```

1. Generate the private key (with no passphrase) with following command:
```
openssl genrsa -out ~/.oci/oci_api_key.pem 2048
```

2. Ensure that only you can read the private key file:
```
chmod go-rwx ~/.oci/oci_api_key.pem
```

> Pipeline needs `api_key` which content is found in `~/.oci/oci_api_key.pm` file, in the following format:

```
		"api_key": "-----BEGIN RSA PRIVATE KEY-----\n.....\n-----END RSA PRIVATE KEY-----\n"
```

3. Generate the public key:
```
openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem
```

4. Copy the contents of the public key to the clipboard using pbcopy
```
cat ~/.oci/oci_api_key_public.pem | pbcopy
```

5. Upload the public key in the console, under user settings.

<p align="center">
<img src="images/oracle_pub_key.png" width="700">
</p>

#### api_key_fingerprint

You can get the key's fingerprint with the following OpenSSL command:
```
openssl rsa -pubout -outform DER -in ~/.oci/oci_api_key.pem | openssl md5 -c
```

## Save Oracle secret in pipeline

Url: `{{url}}/api/v1/orgs/:orgId/secrets`

Body:
```
{
	"name": "my-oci-secret-{{$randomInt}}",
	"type": "oracle",
	"version": 1,
	"values": {
		"user_ocid": "ocid1.user.oc1.....",
		"api_key_fingerprint": "54:ca:d0:f7:......",
		"api_key": "-----BEGIN RSA PRIVATE KEY-----\n.....\n-----END RSA PRIVATE KEY-----\n",
		"region": "eu-frankfurt-1",
		"compartment_ocid": "ocid1.compartment.oc1.........",
		"tenancy_ocid": "ocid1.tenancy.oc1..........."
	}
}
```
