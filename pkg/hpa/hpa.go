// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hpa

import (
	"errors"
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
)

type ValueType string

// nolint: gochecknoglobals
var (
	// PercentageValueType specifies that value is a percentage
	PercentageValueType ValueType = "percentage"
	// QuantityValueType specifies that value is a K8s quantity
	QuantityValueType ValueType = "quantity"
)

type ResourceMetric struct {
	TargetAverageValueType ValueType `json:"targetAverageValueType,omitempty"`
	TargetAverageValue     string    `json:"targetAverageValue,omitempty"`
}

type ResourceMetricStatus struct {
	ResourceMetric
	CurrentAverageValueType ValueType `json:"currentAverageUtilization,omitempty"`
	CurrentAverageValue     string    `json:"currentAverageValue,omitempty"`
}

type CustomMetric struct {
	Query              string `json:"query"`
	TargetValue        string `json:"targetValue,omitempty"`
	TargetAverageValue string `json:"targetAverageValue,omitempty"`
}

type CustomMetricStatus struct {
	CustomMetric
	CurrentValue string `json:"currentValue,omitempty"`
}

type DeploymentScaleStatus struct {
	CurrentReplicas int32  `json:"currentReplicas,omitempty"`
	DesiredReplicas int32  `json:"desiredReplicas,omitempty"`
	Message         string `json:"message,omitempty"`
}

type DeploymentScalingRequest struct {
	ScaleTarget   string                  `json:"scaleTarget"`
	MinReplicas   int32                   `json:"minReplicas"`
	MaxReplicas   int32                   `json:"maxReplicas"`
	Cpu           ResourceMetric          `json:"cpu,omitempty"`
	Memory        ResourceMetric          `json:"memory,omitempty"`
	CustomMetrics map[string]CustomMetric `json:"customMetrics,omitempty"`
}

func (r *DeploymentScalingRequest) Validate() error {
	if r.MaxReplicas <= r.MinReplicas {
		return errors.New("'maxReplicas' should be greater then 'minReplicas'")
	}
	metricCount := 0
	if len(r.Cpu.TargetAverageValueType) != 0 {
		err := r.Cpu.validateResourceMetric()
		if err != nil {
			return err
		}
		metricCount++
	}
	if len(r.Memory.TargetAverageValueType) != 0 {
		err := r.Memory.validateResourceMetric()
		if err != nil {
			return err
		}
		metricCount++
	}
	for _, cm := range r.CustomMetrics {
		err := cm.validateCustomMetric()
		if err != nil {
			return err
		}
		metricCount++
	}
	if metricCount == 0 {
		return errors.New("there should at least one cpu / memory or custom metric specified")
	}
	return nil
}

func (rm ResourceMetric) validateResourceMetric() error {
	switch rm.TargetAverageValueType {
	case PercentageValueType:
		int64Value, err := strconv.ParseInt(rm.TargetAverageValue, 10, 32)
		if err != nil {
			return errors.New("invalid percentage value specified")
		}
		targetValue := int32(int64Value)
		if targetValue <= 0 || targetValue > 100 {
			return fmt.Errorf("invalid percentage value specified: %v (Percentage value shoud be between [1,99]", targetValue)

		}
	case QuantityValueType:
		_, err := resource.ParseQuantity(rm.TargetAverageValue)
		if err != nil {
			return fmt.Errorf("invalid resource metric value: %v (%v)", rm.TargetAverageValue, err.Error())
		}
	}
	return nil
}

func (rm CustomMetric) validateCustomMetric() error {
	if len(rm.Query) == 0 {
		return fmt.Errorf("query is required for custom metric")
	}
	if len(rm.TargetValue) > 0 {
		_, err := resource.ParseQuantity(rm.TargetValue)
		if err != nil {
			return fmt.Errorf("invalid custom metric targetValue: %s (%s)", rm.TargetValue, err.Error())
		}
	} else if len(rm.TargetAverageValue) > 0 {
		_, err := resource.ParseQuantity(rm.TargetAverageValue)
		if err != nil {
			return fmt.Errorf("invalid custom metric targetAverageValue: %s (%s)", rm.TargetAverageValue, err.Error())
		}
	} else {
		return errors.New("either targetValue or targetAverageValue is required")
	}

	return nil
}

type DeploymentScalingInfo struct {
	ScaleTarget   string                        `json:"scaleTarget,omitempty"`
	Kind          string                        `json:"kind,omitempty"`
	MinReplicas   int32                         `json:"minReplicas,omitempty"`
	MaxReplicas   int32                         `json:"maxReplicas,omitempty"`
	Cpu           ResourceMetricStatus          `json:"cpu,omitempty"`
	Memory        ResourceMetricStatus          `json:"memory,omitempty"`
	CustomMetrics map[string]CustomMetricStatus `json:"customMetrics,omitempty"`
	Status        DeploymentScaleStatus         `json:"status,omitempty"`
}
