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
	"fmt"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	credentialprovider "github.com/vdemeester/k8s-pkg-credentialprovider"
	credentialprovidersecrets "github.com/vdemeester/k8s-pkg-credentialprovider/secrets"
	v1 "k8s.io/api/core/v1"
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

var (
	keyring credentialprovider.DockerKeyring
	once    sync.Once
)

// New returns a new authn.Keychain suitable for resolving image references as
// scoped by the provided Options.  It speaks to Kubernetes through the provided
// client interface.
func New(client kubernetes.Interface, opt Options) (authn.Keychain, error) {
	if opt.Namespace == "" {
		opt.Namespace = "default"
	}
	if opt.ServiceAccountName == "" {
		opt.ServiceAccountName = "default"
	}

	// Implement a Kubernetes-style authentication keychain.
	// This needs to support roughly the following kinds of authentication:
	//  1) The implicit authentication from k8s.io/kubernetes/pkg/credentialprovider
	//  2) The explicit authentication from imagePullSecrets on Pod
	//  3) The semi-implicit authentication where imagePullSecrets are on the
	//    Pod's service account.

	// First, fetch all of the explicitly declared pull secrets
	var pullSecrets []v1.Secret
	if client != nil {
		for _, name := range opt.ImagePullSecrets {
			ps, err := client.CoreV1().Secrets(opt.Namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			pullSecrets = append(pullSecrets, *ps)
		}

		// Second, fetch all of the pull secrets attached to our service account.
		sa, err := client.CoreV1().ServiceAccounts(opt.Namespace).Get(opt.ServiceAccountName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		for _, localObj := range sa.ImagePullSecrets {
			ps, err := client.CoreV1().Secrets(opt.Namespace).Get(localObj.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			pullSecrets = append(pullSecrets, *ps)
		}
	}

	once.Do(func() {
		keyring = credentialprovider.NewDockerKeyring()
	})

	// Third, extend the default keyring with the pull secrets.
	kr, err := credentialprovidersecrets.MakeDockerKeyring(pullSecrets, keyring)
	if err != nil {
		return nil, err
	}
	return &keychain{
		keyring: kr,
	}, nil
}

// NewInCluster returns a new authn.Keychain suitable for resolving image references as
// scoped by the provided Options, constructing a kubernetes.Interface based on in-cluster
// authentication.
func NewInCluster(opt Options) (authn.Keychain, error) {
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}
	return New(client, opt)
}

// NewNoClient returns a new authn.Keychain that supports the portions of the K8s keychain
// that don't read ImagePullSecrets.  This limits it to roughly the Node-identity-based
// authentication schemes in Kubernetes pkg/credentialprovider.  This version of the
// k8schain drops the requirement that we run as a K8s serviceaccount with access to all
// of the on-cluster secrets.  This drop in fidelity also diminishes its value as a stand-in
// for Kubernetes authentication, but this actually targets a different use-case.  What
// remains is an interesting sweet spot: this variant can serve as a credential provider
// for all of the major public clouds, but in library form (vs. an executable you exec).
func NewNoClient() (authn.Keychain, error) {
	return New(nil, Options{})
}

type lazyProvider struct {
	kc    *keychain
	image string
}

// Authorization implements Authenticator.
func (lp lazyProvider) Authorization() (*authn.AuthConfig, error) {
	creds, found := lp.kc.keyring.Lookup(lp.image)
	if !found || len(creds) < 1 {
		return nil, fmt.Errorf("keychain returned no credentials for %q", lp.image)
	}
	authConfig := creds[0]
	return &authn.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}, nil
}

type keychain struct {
	keyring credentialprovider.DockerKeyring
}

// Resolve implements authn.Keychain
func (kc *keychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	var image string
	if repo, ok := target.(name.Repository); ok {
		image = repo.String()
	} else {
		// Lookup expects an image reference and we only have a registry.
		image = target.RegistryStr() + "/foo/bar"
	}

	if creds, found := kc.keyring.Lookup(image); !found || len(creds) < 1 {
		return authn.Anonymous, nil
	}
	// TODO(mattmoor): How to support multiple credentials?
	return lazyProvider{
		kc:    kc,
		image: image,
	}, nil
}
