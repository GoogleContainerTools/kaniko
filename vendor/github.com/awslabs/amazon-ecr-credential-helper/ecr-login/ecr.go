// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package ecr

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/docker/docker-credential-helpers/credentials"
)

var notImplemented = errors.New("not implemented")

type ECRHelper struct {
	ClientFactory api.ClientFactory
}

// ensure ECRHelper adheres to the credentials.Helper interface
var _ credentials.Helper = (*ECRHelper)(nil)

func (ECRHelper) Add(creds *credentials.Credentials) error {
	// This does not seem to get called
	return notImplemented
}

func (ECRHelper) Delete(serverURL string) error {
	// This does not seem to get called
	return notImplemented
}

func (self ECRHelper) Get(serverURL string) (string, string, error) {
	registry, err := api.ExtractRegistry(serverURL)
	if err != nil {
		logrus.
			WithError(err).
			WithField("serverURL", serverURL).
			Error("Error parsing the serverURL")
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	var client api.Client
	if registry.FIPS {
		client, err = self.ClientFactory.NewClientWithFipsEndpoint(registry.Region)
		if err != nil {
			logrus.WithError(err).Error("Error resolving FIPS endpoint")
			return "", "", credentials.NewErrCredentialsNotFound()
		}
	} else {
		client = self.ClientFactory.NewClientFromRegion(registry.Region)
	}

	auth, err := client.GetCredentials(serverURL)
	if err != nil {
		logrus.WithError(err).Error("Error retrieving credentials")
		return "", "", credentials.NewErrCredentialsNotFound()
	}
	return auth.Username, auth.Password, nil
}

func (self ECRHelper) List() (map[string]string, error) {
	logrus.Debug("Listing credentials")
	client := self.ClientFactory.NewClientWithDefaults()

	auths, err := client.ListCredentials()
	if err != nil {
		logrus.WithError(err).Error("Error listing credentials")
		return nil, fmt.Errorf("ecr: could not list credentials: %v", err)
	}

	result := map[string]string{}

	for _, auth := range auths {
		serverURL := auth.ProxyEndpoint
		result[serverURL] = auth.Username
	}
	return result, nil
}
