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
	"context"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	kauth "github.com/google/go-containerregistry/pkg/authn/kubernetes"
	"github.com/google/go-containerregistry/pkg/v1/google"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	amazonKeychain authn.Keychain = authn.NewKeychainFromHelper(ecr.ECRHelper{ClientFactory: api.DefaultClientFactory{}})
	azureKeychain  authn.Keychain = authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
)

// Options holds configuration data for guiding credential resolution.
type Options = kauth.Options

// New returns a new authn.Keychain suitable for resolving image references as
// scoped by the provided Options.  It speaks to Kubernetes through the provided
// client interface.
func New(ctx context.Context, client kubernetes.Interface, opt Options) (authn.Keychain, error) {
	k8s, err := kauth.New(ctx, client, kauth.Options(opt))
	if err != nil {
		return nil, err
	}

	return authn.NewMultiKeychain(
		authn.DefaultKeychain,
		google.Keychain,
		amazonKeychain,
		azureKeychain,
		k8s,
	), nil
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

// NewNoClient returns a new authn.Keychain that supports the portions of the K8s keychain
// that don't read ImagePullSecrets.  This limits it to roughly the Node-identity-based
// authentication schemes in Kubernetes pkg/credentialprovider.  This version of the
// k8schain drops the requirement that we run as a K8s serviceaccount with access to all
// of the on-cluster secrets.  This drop in fidelity also diminishes its value as a stand-in
// for Kubernetes authentication, but this actually targets a different use-case.  What
// remains is an interesting sweet spot: this variant can serve as a credential provider
// for all of the major public clouds, but in library form (vs. an executable you exec).
func NewNoClient(ctx context.Context) (authn.Keychain, error) {
	return authn.NewMultiKeychain(
		authn.DefaultKeychain,
		google.Keychain,
		amazonKeychain,
		azureKeychain,
	), nil
}

// NewFromPullSecrets returns a new authn.Keychain suitable for resolving image references as
// scoped by the pull secrets.
func NewFromPullSecrets(ctx context.Context, pullSecrets []corev1.Secret) (authn.Keychain, error) {
	k8s, err := kauth.NewFromPullSecrets(ctx, pullSecrets)
	if err != nil {
		return nil, err
	}

	return authn.NewMultiKeychain(
		authn.DefaultKeychain,
		google.Keychain,
		amazonKeychain,
		azureKeychain,
		k8s,
	), nil
}
