/*
Copyright © 2020 Chris Mellard chris.mellard@icloud.com

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
package token

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func GetServicePrincipalTokenFromEnvironment() (*adal.ServicePrincipalToken, auth.EnvironmentSettings, error) {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return &adal.ServicePrincipalToken{}, auth.EnvironmentSettings{}, fmt.Errorf("failed to get auth settings from environment - %w", err)
	}

	spToken, err := getServicePrincipalToken(settings, settings.Environment.ResourceManagerEndpoint)
	if err != nil {
		return &adal.ServicePrincipalToken{}, auth.EnvironmentSettings{}, fmt.Errorf("failed to initialise sp token config %w", err)
	}

	return spToken, settings, nil
}

// getServicePrincipalToken retrieves an Azure AD OAuth2 token from the supplied environment settings for the specified resource
func getServicePrincipalToken(settings auth.EnvironmentSettings, resource string) (*adal.ServicePrincipalToken, error) {

	//1.Client Credentials
	if _, e := settings.GetClientCredentials(); e == nil {
		clientCredentialsConfig, err := settings.GetClientCredentials()
		if err != nil {
			return &adal.ServicePrincipalToken{}, fmt.Errorf("failed to get client credentials settings from environment - %w", err)
		}
		oAuthConfig, err := adal.NewOAuthConfig(settings.Environment.ActiveDirectoryEndpoint, clientCredentialsConfig.TenantID)
		if err != nil {
			return &adal.ServicePrincipalToken{}, fmt.Errorf("failed to initialise OAuthConfig - %w", err)
		}
		return adal.NewServicePrincipalToken(*oAuthConfig, clientCredentialsConfig.ClientID, clientCredentialsConfig.ClientSecret, clientCredentialsConfig.Resource)
	}

	//2. Client Certificate
	if _, e := settings.GetClientCertificate(); e == nil {
		return &adal.ServicePrincipalToken{}, fmt.Errorf("authentication method currently unsupported")
	}

	//3. Username Password
	if _, e := settings.GetUsernamePassword(); e == nil {
		return &adal.ServicePrincipalToken{}, fmt.Errorf("authentication method currently unsupported")
	}

	// 4. MSI
	msiEndpoint, err := adal.GetMSIEndpoint()
	if err != nil {
		return &adal.ServicePrincipalToken{}, fmt.Errorf("unable to determine MSIEndpoint %w", err)
	}

	return adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
}
