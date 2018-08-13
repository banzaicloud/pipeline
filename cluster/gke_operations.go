package cluster

type OperationInfoer interface {
	GetInfo(projectId, location, operationName string) (status string, opType string, err error)
}
