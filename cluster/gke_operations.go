package cluster

type OperationInfoer interface {
	GetInfo(operationName string) (status string, opType string, err error)
}
