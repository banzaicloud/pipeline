module github.com/banzaicloud/pipeline/pkg/sdk

go 1.14

require (
	emperror.dev/errors v0.7.0
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/golang/mock v1.4.3 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/sirupsen/logrus v1.5.0 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/uber/tchannel-go v1.18.0 // indirect
	go.uber.org/cadence v0.9.0
	go.uber.org/thriftrw v1.23.0 // indirect
	go.uber.org/yarpc v1.45.0 // indirect
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/net v0.0.0-20200421231249-e086a090c8fd // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
)

replace (
	github.com/apache/thrift => github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08
)
