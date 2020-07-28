// Copyright © 2020 Banzai Cloud
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

package monitoring

import (
	"context"
	"testing"

	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMigrateIngress(t *testing.T) {
	releaseNamespace := "asd"
	ingressNameToRemove := "aaa"
	oldVersion := "8.5.14"
	newVersion := "8.13.8"

	allIngresses := []runtime.Object{
		&v1beta1.Ingress{
			ObjectMeta: v1.ObjectMeta{
				Name:      ingressNameToRemove,
				Namespace: releaseNamespace,
				Labels: map[string]string{
					"release": prometheusOperatorReleaseName,
				},
			},
		},
		&v1beta1.Ingress{
			ObjectMeta: v1.ObjectMeta{
				Name:      "bbb",
				Namespace: releaseNamespace,
				Labels: map[string]string{
					"does": "notmatch",
				},
			},
		},
	}

	t.Run("migrate", func(t *testing.T) {
		clientset := fake.NewSimpleClientset(allIngresses...)

		clientsetFactory := func() (kubernetes.Interface, error) {
			return clientset, nil
		}

		err := Migrate(context.Background(), clientsetFactory, releaseNamespace, oldVersion, newVersion)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		ingresses, err := clientset.ExtensionsV1beta1().Ingresses(releaseNamespace).List(context.Background(), v1.ListOptions{})
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(ingresses.Items) != len(allIngresses)-1 {
			t.Fatalf("invalid number of ingresses left, expected %d, got %d: %+v", len(allIngresses)-1, len(ingresses.Items), ingresses.Items)
		}

		for _, i := range ingresses.Items {
			if i.Name == ingressNameToRemove && i.Namespace == releaseNamespace {
				t.Fatal("failed to remove the single ingress expected")
			}
		}
	})

	t.Run("nomigrate", func(t *testing.T) {
		clientset := fake.NewSimpleClientset(allIngresses...)

		clientsetFactory := func() (kubernetes.Interface, error) {
			return clientset, nil
		}

		err := Migrate(context.Background(), clientsetFactory, releaseNamespace, oldVersion, oldVersion)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		ingresses, err := clientset.ExtensionsV1beta1().Ingresses(releaseNamespace).List(context.Background(), v1.ListOptions{})
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(ingresses.Items) != len(allIngresses) {
			t.Fatalf("expected that nobody get harmed")
		}
	})
}
