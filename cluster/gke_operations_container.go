package cluster

import (
	"context"

	gke "google.golang.org/api/container/v1"
)

type containerOperation struct {
	svc *gke.Service
}

func (co *containerOperation) GetInfo(projectId, zone, operationName string) (string, string, error) {
	op, err := co.svc.Projects.Zones.Operations.Get(projectId, zone, operationName).Context(context.Background()).Do()
	if err != nil {
		return "", "", err
	}

	return op.Status, op.OperationType, nil
}

func newContainerOperation(svc *gke.Service) *containerOperation {
	return &containerOperation{
		svc: svc,
	}
}
