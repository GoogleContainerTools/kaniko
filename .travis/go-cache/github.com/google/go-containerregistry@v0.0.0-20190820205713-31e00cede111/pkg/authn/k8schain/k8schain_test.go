// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8schain

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "k8s.io/client-go/kubernetes/fake"
)

func TestAnonymousFallback(t *testing.T) {
	client := fakeclient.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
	})

	kc, err := New(client, Options{})
	if err != nil {
		t.Errorf("New() = %v", err)
	}

	reg, err := name.NewRegistry("fake.registry.io", name.WeakValidation)
	if err != nil {
		t.Errorf("NewRegistry() = %v", err)
	}

	auth, err := kc.Resolve(reg)
	if err != nil {
		t.Errorf("Resolve(%v) = %v", reg, err)
	}
	if got, want := auth, authn.Anonymous; got != want {
		t.Errorf("Resolve() = %v, want %v", got, want)
	}
}

func TestAttachedServiceAccount(t *testing.T) {
	username, password := "foo", "bar"
	client := fakeclient.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svcacct",
			Namespace: "ns",
		},
		ImagePullSecrets: []corev1.LocalObjectReference{{
			Name: "secret",
		}},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeDockercfg,
		Data: map[string][]byte{
			corev1.DockerConfigKey: []byte(
				fmt.Sprintf(`{"fake.registry.io": {"username": "%s", "password": "%s"}}`,
					username, password),
			),
		},
	})

	kc, err := New(client, Options{
		Namespace:          "ns",
		ServiceAccountName: "svcacct",
	})
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	reg, err := name.NewRegistry("fake.registry.io", name.WeakValidation)
	if err != nil {
		t.Errorf("NewRegistry() = %v", err)
	}

	auth, err := kc.Resolve(reg)
	if err != nil {
		t.Errorf("Resolve(%v) = %v", reg, err)
	}
	got, err := auth.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}
	want, err := (&authn.Basic{Username: username, Password: password}).Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}
	if got != want {
		t.Errorf("Resolve() = %v, want %v", got, want)
	}
}

func TestImagePullSecrets(t *testing.T) {
	username, password := "foo", "bar"
	client := fakeclient.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ns",
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeDockercfg,
		Data: map[string][]byte{
			corev1.DockerConfigKey: []byte(
				fmt.Sprintf(`{"fake.registry.io": {"auth": "%s"}}`,
					base64.StdEncoding.EncodeToString([]byte(username+":"+password))),
			),
		},
	})

	kc, err := New(client, Options{
		Namespace:        "ns",
		ImagePullSecrets: []string{"secret"},
	})
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	reg, err := name.NewRegistry("fake.registry.io", name.WeakValidation)
	if err != nil {
		t.Errorf("NewRegistry() = %v", err)
	}

	auth, err := kc.Resolve(reg)
	if err != nil {
		t.Errorf("Resolve(%v) = %v", reg, err)
	}
	got, err := auth.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}
	want, err := (&authn.Basic{Username: username, Password: password}).Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}
	if got != want {
		t.Errorf("Resolve() = %v, want %v", got, want)
	}
}
