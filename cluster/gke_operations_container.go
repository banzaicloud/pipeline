package cluster

import (
	"context"

	gke "google.golang.org/api/container/v1"
)

type containerOperation struct {
	svc       *gke.Service
	projectId string
	zone      string
}

func (co *containerOperation) GetInfo(operationName string) (string, string, error) {
	op, err := co.svc.Projects.Zones.Operations.Get(co.projectId, co.zone, operationName).Context(context.Background()).Do()
	if err != nil {
		return "", "", err
	}

	return op.Status, op.OperationType, nil
}

func newContainerOperation(svc *gke.Service, projectId, zone string) OperationInfoer {
	return &containerOperation{
		svc:       svc,
		projectId: projectId,
		zone:      zone,
	}
}
