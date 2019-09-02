# Cluster feature implementation guide
Cluster features are services and behaviors associated with a cluster that can be enabled and disabled independently.

To implement a new cluster feature you should:
1. create a new directory/package under `internal/clusterfeature/features` with the name of your feature
2. implement the `FeatureManager` and the `FeatureOperator` interface in this package
3. register an instance of your manager in the pipeline's `FeatureManagerRegistry` instance and an instance of your operator in the worker's `FeatureOperatorRegistry` instance

## Tips and suggestions
Your feature manager and feature operator should have the same unique `Name()` that is consistent with the package name. Feature names always use `camelCase`.

The two feature operations, `Apply` and `Deactivate` perform their tasks synchronously.
They should return a `ClusterNotReadyError`—or an error implementing the `ShouldRetry() bool` behavior—when the cluster's not (yet) ready for the operation.
The `Apply` method receives its specification run through `PrepareSpec`.

We suggest using the `Transformation`s in `pkg/opaque` when implementing specification transformations in `PrepareSpec`. When frequently used generic patterns arise, please consider factoring them out into the `opaque` package for reuse (or the `internal/clusterfeature/features` package for more specific patterns).

If the original specification provided by the user is considered valid by `ValidateSpec` the prepared specification should be too.

## Example
```go
// internal/clusterfeature/features/example/common.go
package example

const FeatureName = "example"

```

```go
// internal/clusterfeature/features/example/manager.go
package example

import (
    "context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

type FeatureManager struct {
    // <-- Your dependencies here
}

func (m FeatureManager) Name() string {
	return FeatureName
}

func (m FeatureManager) GetOutput(ctx context.Context, clusterID uint) (clusterfeature.FeatureOutput, error) {
    // <-- Do interesting queries and data collection here

    return clusterfeature.FeatureOutput{
        "strKey": "value",
        "intKey": 42,
    }, nil
}

func (m FeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
    if _, ok := spec["importantKey"]; !ok {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     "importantKey is missing",
		}
    }
    return nil
}

func (m FeatureManager) PrepareSpec(ctx context.Context, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
    return spec, nil  // no preparation necessary
}
```

```go
// internal/clusterfeature/features/example/operator.go
package example

import (
    "context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

type FeatureOperator struct {
	clusterService clusterfeature.ClusterService
    // <-- Your dependencies here
}

func (op FeatureOperator) Name() string {
	return FeatureName
}

func (op FeatureOperator) Apply(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
        // Cannot apply spec if cluster is not ready
		return err
	}

    // <-- Apply the new spec here
    return nil
}

func (op FeatureOperator) Deactivate(ctx context.Context, clusterID uint) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
        // Cannot deactivate feature if cluster is not ready
		return err
	}

    // <-- Deactivate the feature here
    return nil
}
```