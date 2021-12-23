/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package creds

import (
	"context"
	"sync"

	"github.com/genuinetools/bpfd/proc"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/sirupsen/logrus"
)

var (
	setupKeychainOnce sync.Once
	keychain          authn.Keychain
)

// GetKeychain returns a keychain for accessing container registries.
func GetKeychain() authn.Keychain {
	setupKeychainOnce.Do(func() {
		keychain = authn.DefaultKeychain

		// Historically kaniko was pre-configured by default with gcr
		// credential helper, in here we keep the backwards
		// compatibility by enabling the GCR helper only when gcr.io
		// (or pkg.dev) is in one of the destinations.
		gauth, err := google.NewEnvAuthenticator()
		if err != nil {
			logrus.Warnf("Failed to setup Google env authenticator, ignoring: %v", err)
		} else {
			keychain = authn.NewMultiKeychain(authn.DefaultKeychain, gcrKeychain{gauth})
		}

		// Add the Kubernetes keychain if we're on Kubernetes
		if proc.GetContainerRuntime(0, 0) == proc.RuntimeKubernetes {
			k8sc, err := k8schain.NewNoClient(context.Background())
			if err != nil {
				logrus.Warnf("Error setting up k8schain. Using default keychain %s", err)
				return
			}
			keychain = authn.NewMultiKeychain(keychain, k8sc)
		}
	})
	return keychain
}
