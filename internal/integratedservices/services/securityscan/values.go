// Copyright © 2019 Banzai Cloud
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

package securityscan

// represents a values yaml to be passed to the anchore image validator webhook chart
type ImageValidatorChartValues struct {
	ExternalAnchore   *AnchoreValues     `json:"externalAnchore,omitempty" mapstructure:"externalAnchore"`
	NamespaceSelector *NamespaceSelector `json:"namespaceSelector,omitempty" mapstructure:"namespaceSelector"`
	ObjectSelector    *ObjectSelector    `json:"objectSelector,omitempty" mapstructure:"objectSelector"`
}

// AnchoreValues struct used to build chart values and to extract anchore data from secret values
type AnchoreValues struct {
	Host     string `json:"anchoreHost" mapstructure:"host"`
	User     string `json:"anchoreUser" mapstructure:"username"`
	Password string `json:"anchorePass" mapstructure:"password"`
}

type MatchExpression struct {
	Key      string   `json:"key" mapstructure:"key"`
	Operator string   `json:"operator" mapstructure:"operator"`
	Values   []string `json:"values" mapstructure:"values"`
}

type NamespaceSelector struct {
	MatchLabels      map[string]string `json:"matchLabels,omitempty" mapstructure:"matchLabels"`
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty" mapstructure:"matchExpressions"`
}

type ObjectSelector struct {
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty" mapstructure:"matchExpressions"`
}

func (n *NamespaceSelector) addMatchLabel(key string, value string) {
	if n.MatchLabels == nil {
		n.MatchLabels = make(map[string]string)
	}
	n.MatchLabels[key] = value
}

func (n NamespaceSelector) addMatchExpression(key string, operator string, values []string) {
	if n.MatchExpressions == nil {
		n.MatchExpressions = make([]MatchExpression, 0, len(values))
	}

	n.MatchExpressions = append(n.MatchExpressions,
		MatchExpression{
			Key:      key,
			Operator: operator,
			Values:   values,
		})
}

func (o ObjectSelector) addMatchExpression(key string, operator string, values []string) {
	if o.MatchExpressions == nil {
		o.MatchExpressions = make([]MatchExpression, 0, len(values))
	}

	o.MatchExpressions = append(o.MatchExpressions,
		MatchExpression{
			Key:      key,
			Operator: operator,
			Values:   values,
		})
}
