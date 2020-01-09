# Integrated service implementation guide
Cluster integrated services are services and behaviors associated with a cluster that can be enabled and disabled independently.

To implement a new integrated service you should:
1. create a new directory/package under `internal/integratedservices/services` with the name of your service
2. implement the `IntegratedServiceManager` and the `IntegratedServiceOperator` interface in this package
3. register an instance of your manager in the pipeline's `IntegratedServiceManagerRegistry` instance and an instance of your operator in the worker's `IntegratedServiceOperatorRegistry` instance

## Tips and suggestions
Your service manager and service operator should have the same unique `Name()` that is consistent with the package name. Service names always use `camelCase`.

The two service operations, `Apply` and `Deactivate` perform their tasks synchronously.
They should return a `ClusterNotReadyError`—or an error implementing the `ShouldRetry() bool` behavior—when the cluster's not (yet) ready for the operation.
The `Apply` method receives its specification run through `PrepareSpec`.

We suggest using the `Transformation`s in `pkg/any` when implementing specification transformations in `PrepareSpec`. When frequently used generic patterns arise, please consider factoring them out into the `any` package for reuse (or the `internal/integratedservices/services/` package for more specific patterns).

If the original specification provided by the user is considered valid by `ValidateSpec` the prepared specification should be too.

## Example
```go
// internal/integratedservices/services/example/common.go
package example

const integratedServiceName = "example"

```

```go
// internal/integratedservices/services/example/manager.go
package example

import (
    "context"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type IntegratedServiceManager struct {
    // <-- Your dependencies here
}

func (m IntegratedServiceManager) Name() string {
	return integratedServiceName
}

func (m IntegratedServiceManager) GetOutput(ctx context.Context, clusterID uint) (integratedservices.IntegratedServiceOutput, error) {
    // <-- Do interesting queries and data collection here

    return integratedservices.IntegratedServiceOutput{
        "strKey": "value",
        "intKey": 42,
    }, nil
}

func (m IntegratedServiceManager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
    if _, ok := spec["importantKey"]; !ok {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: integratedServiceName,
			Problem:     "importantKey is missing",
		}
    }
    return nil
}

func (m IntegratedServiceManager) PrepareSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceSpec, error) {
    return spec, nil  // no preparation necessary
}
```

```go
// internal/integratedservices/services/example/operator.go
package example

import (
    "context"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type IntegratedServiceOperator struct {
	clusterService integratedservices.ClusterService
    // <-- Your dependencies here
}

func (op IntegratedServiceOperator) Name() string {
	return integratedServiceName
}

func (op IntegratedServiceOperator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
        // Cannot apply spec if cluster is not ready
		return err
	}

    // <-- Apply the new spec here
    return nil
}

func (op IntegratedServiceOperator) Deactivate(ctx context.Context, clusterID uint) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
        // Cannot deactivate service if cluster is not ready
		return err
	}

    // <-- Deactivate the service here
    return nil
}
```
