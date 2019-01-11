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
	"sync"

	"github.com/genuinetools/amicontained/container"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/sirupsen/logrus"
)

var (
	setupKeyChainOnce sync.Once
	keyChain          authn.Keychain
)

// GetKeychain returns a keychain for accessing container registries.
func GetKeychain() authn.Keychain {
	setupKeyChainOnce.Do(func() {
		keyChain = authn.NewMultiKeychain(authn.DefaultKeychain)

		// Add the Kubernetes keychain if we're on Kubernetes
		r, err := container.DetectRuntime()
		if err != nil {
			logrus.Warnf("Error detecting container runtime. Using default keychain: %s", err)
			return
		}
		if r == container.RuntimeKubernetes {
			k8sc, err := k8schain.NewNoClient()
			if err != nil {
				logrus.Warnf("Error setting up k8schain. Using default keychain %s", err)
				return
			}
			keyChain = authn.NewMultiKeychain(keyChain, k8sc)
		}
	})
	return keyChain
}
