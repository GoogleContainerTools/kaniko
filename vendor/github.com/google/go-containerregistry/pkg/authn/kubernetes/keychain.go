// Copyright 2022 Google LLC All Rights Reserved.
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

package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Options holds configuration data for guiding credential resolution.
type Options struct {
	// Namespace holds the namespace inside of which we are resolving the
	// image reference.  If empty, "default" is assumed.
	Namespace string
	// ServiceAccountName holds the serviceaccount as which the container
	// will run (scoped to Namespace).  If empty, "default" is assumed.
	ServiceAccountName string
	// ImagePullSecrets holds the names of the Kubernetes secrets (scoped to
	// Namespace) containing credential data to use for the image pull.
	ImagePullSecrets []string
}

// New returns a new authn.Keychain suitable for resolving image references as
// scoped by the provided Options.  It speaks to Kubernetes through the provided
// client interface.
func New(ctx context.Context, client kubernetes.Interface, opt Options) (authn.Keychain, error) {
	if opt.Namespace == "" {
		opt.Namespace = "default"
	}
	if opt.ServiceAccountName == "" {
		opt.ServiceAccountName = "default"
	}

	// Implement a Kubernetes-style authentication keychain.
	// This needs to support roughly the following kinds of authentication:
	//  1) The explicit authentication from imagePullSecrets on Pod
	//  2) The semi-implicit authentication where imagePullSecrets are on the
	//    Pod's service account.

	// First, fetch all of the explicitly declared pull secrets
	var pullSecrets []corev1.Secret
	for _, name := range opt.ImagePullSecrets {
		ps, err := client.CoreV1().Secrets(opt.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pullSecrets = append(pullSecrets, *ps)
	}

	// Second, fetch all of the pull secrets attached to our service account.
	sa, err := client.CoreV1().ServiceAccounts(opt.Namespace).Get(ctx, opt.ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	for _, localObj := range sa.ImagePullSecrets {
		ps, err := client.CoreV1().Secrets(opt.Namespace).Get(ctx, localObj.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pullSecrets = append(pullSecrets, *ps)
	}

	return NewFromPullSecrets(ctx, pullSecrets)
}

// NewInCluster returns a new authn.Keychain suitable for resolving image references as
// scoped by the provided Options, constructing a kubernetes.Interface based on in-cluster
// authentication.
func NewInCluster(ctx context.Context, opt Options) (authn.Keychain, error) {
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}
	return New(ctx, client, opt)
}

// NewFromPullSecrets returns a new authn.Keychain suitable for resolving image references as
// scoped by the pull secrets.
func NewFromPullSecrets(ctx context.Context, secrets []corev1.Secret) (authn.Keychain, error) {
	m := map[string]authn.AuthConfig{}
	for _, secret := range secrets {
		auths := map[string]authn.AuthConfig{}
		if b, exists := secret.Data[corev1.DockerConfigJsonKey]; secret.Type == corev1.SecretTypeDockerConfigJson && exists && len(b) > 0 {
			var cfg struct {
				Auths map[string]authn.AuthConfig
			}
			if err := json.Unmarshal(b, &cfg); err != nil {
				return nil, err
			}
			auths = cfg.Auths
		}
		if b, exists := secret.Data[corev1.DockerConfigKey]; secret.Type == corev1.SecretTypeDockercfg && exists && len(b) > 0 {
			if err := json.Unmarshal(b, &auths); err != nil {
				return nil, err
			}
		}

		for k, v := range auths {
			// Don't overwrite previously specified Auths for a
			// given key.
			if _, found := m[k]; !found {
				m[k] = v
			}
		}
	}
	return authsKeychain(m), nil
}

type authsKeychain map[string]authn.AuthConfig

func (kc authsKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	// Check for an auth that matches the repository, then if that's not
	// found, one that matches the registry.
	var cfg authn.AuthConfig
	for _, key := range []string{target.String(), target.RegistryStr()} {
		if key == name.DefaultRegistry {
			key = authn.DefaultAuthKey
		}
		var ok bool
		cfg, ok = kc[key]
		if ok {
			break
		}
	}
	empty := authn.AuthConfig{}
	if cfg == empty {
		return authn.Anonymous, nil
	}

	if cfg.Auth != "" {
		dec, err := base64.StdEncoding.DecodeString(cfg.Auth)
		if err != nil {
			return nil, err
		}
		parts := strings.SplitN(string(dec), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unable to parse auth field, must be formatted as base64(username:password)")
		}
		cfg.Username = parts[0]
		cfg.Password = parts[1]
		cfg.Auth = ""
	}

	return authn.FromConfig(authn.AuthConfig(cfg)), nil
}
