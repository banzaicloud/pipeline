package cluster

import (
	"context"

	gkeCompute "google.golang.org/api/compute/v1"
)

type computeOperation struct {
	csv *gkeCompute.Service
}

func (co *computeOperation) GetInfo(projectId, _, operationName string) (string, string, error) {

	op, err := co.csv.GlobalOperations.Get(projectId, operationName).Context(context.Background()).Do()
	if err != nil {
		return "", "", err
	}

	return op.Status, op.OperationType, nil
}

func newComputeOperation(csv *gkeCompute.Service) *computeOperation {
	return &computeOperation{
		csv: csv,
	}
}
