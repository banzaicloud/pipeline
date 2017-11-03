package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IAMInstanceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Name              string `json:"name,omitempty"`
	Identifier        string `json:"identifier,omitempty"`
	Role              *IAMRole `json:"role,omitempty"`
}

type IAMRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Name              string `json:"name,omitempty"`
	Identifier        string `json:"identifier,omitempty"`
	Policies          []*IAMPolicy `json:"policies,omitempty"`
}

type IAMPolicy struct {
	Name          string `json:"name,omitempty"`
	Document      string `json:"document,omitempty"`
	Identifier    string `json:"identifier,omitempty"`
}


