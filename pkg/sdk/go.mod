module github.com/banzaicloud/pipeline/pkg/sdk

go 1.16

require (
	emperror.dev/errors v0.7.0
	github.com/apache/thrift v0.13.0 // indirect
	github.com/aws/aws-sdk-go v1.34.4
	github.com/mitchellh/mapstructure v1.1.2
	github.com/stretchr/testify v1.6.1
	github.com/uber/tchannel-go v1.18.0 // indirect
	go.uber.org/cadence v0.19.0
	go.uber.org/zap v1.14.1 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.9
	k8s.io/apimachinery v0.21.9
	k8s.io/kubectl v0.21.9
)

replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08
