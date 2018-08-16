package cluster

type operationInfoer interface {
	getInfo(operationName string) (status string, opType string, err error)
}
