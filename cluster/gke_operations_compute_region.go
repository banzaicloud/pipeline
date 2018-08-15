package cluster

import (
	"context"

	gkeCompute "google.golang.org/api/compute/v1"
)

type computeRegionOperation struct {
	csv       *gkeCompute.Service
	projectId string
	region    string
}

func (co *computeRegionOperation) GetInfo(operationName string) (string, string, error) {

	op, err := co.csv.RegionOperations.Get(co.projectId, co.region, operationName).Context(context.Background()).Do()
	if err != nil {
		return "", "", err
	}

	return op.Status, op.OperationType, nil
}

func newComputeRegionOperation(csv *gkeCompute.Service, projectId, region string) OperationInfoer {
	return &computeRegionOperation{
		csv:       csv,
		projectId: projectId,
		region:    region,
	}
}
