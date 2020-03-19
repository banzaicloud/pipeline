# Generate Pipeline certificates with CFSSL

Generating server (Pipeline) and client (Worker) certificates with [CFSSL](https://github.com/cloudflare/cfssl):

```bash
cd config/certs

cfssl genkey -initca csr.json | cfssljson -bare ca

cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=config.json -profile=server certificate.json | cfssljson -bare server

cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=config.json -profile=client certificate.json | cfssljson -bare client

# This example is for the integration test only for testing a bad certificate (with wrong CN)
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=config.json -profile=client certificate-non-pipeline.json | cfssljson -bare client-non-pipeline

rm *.csr
```
