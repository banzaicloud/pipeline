package cluster

import (
	"context"

	gkeCompute "google.golang.org/api/compute/v1"
)

type computeGlobalOperation struct {
	csv       *gkeCompute.Service
	projectId string
}

func (co *computeGlobalOperation) GetInfo(operationName string) (string, string, error) {

	op, err := co.csv.GlobalOperations.Get(co.projectId, operationName).Context(context.Background()).Do()
	if err != nil {
		return "", "", err
	}

	return op.Status, op.OperationType, nil
}

func newComputeGlobalOperation(csv *gkeCompute.Service, projectId string) OperationInfoer {
	return &computeGlobalOperation{
		csv:       csv,
		projectId: projectId,
	}
}
