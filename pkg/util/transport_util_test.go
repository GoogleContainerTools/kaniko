/*
Copyright 2020 Google LLC

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

package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
)

type mockedCertPool struct {
	certificatesPath []string
}

func (m *mockedCertPool) value() *x509.CertPool {
	return &x509.CertPool{}
}

func (m *mockedCertPool) append(path string) error {
	m.certificatesPath = append(m.certificatesPath, path)
	return nil
}

type mockedKeyPairLoader struct {
}

func (p *mockedKeyPairLoader) load(certFile, keyFile string) (tls.Certificate, error) {
	foo := tls.Certificate{}
	return foo, nil
}

func Test_makeTransport(t *testing.T) {
	registryName := "my.registry.name"

	tests := []struct {
		name  string
		opts  config.RegistryOptions
		check func(*tls.Config, *mockedCertPool, error)
	}{
		{
			name: "SkipTLSVerify set",
			opts: config.RegistryOptions{SkipTLSVerify: true},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if !config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify not set while SkipTLSVerify set")
				}
			},
		},
		{
			name: "SkipTLSVerifyRegistries set with expected registry",
			opts: config.RegistryOptions{SkipTLSVerifyRegistries: []string{registryName}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if !config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify not set while SkipTLSVerifyRegistries set with registry name")
				}
			},
		},
		{
			name: "SkipTLSVerifyRegistries set with other registry",
			opts: config.RegistryOptions{SkipTLSVerifyRegistries: []string{fmt.Sprintf("other.%s", registryName)}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify set while SkipTLSVerifyRegistries not set with registry name")
				}
			},
		},
		{
			name: "RegistriesCertificates set for registry",
			opts: config.RegistryOptions{RegistriesCertificates: map[string]string{registryName: "/path/to/the/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if len(pool.certificatesPath) != 1 || pool.certificatesPath[0] != "/path/to/the/certificate.cert" {
					t.Errorf("makeTransport().RegistriesCertificates certificate not appended to system certificates")
				}
			},
		},
		{
			name: "RegistriesCertificates set for another registry",
			opts: config.RegistryOptions{RegistriesCertificates: map[string]string{fmt.Sprintf("other.%s=", registryName): "/path/to/the/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if len(pool.certificatesPath) != 0 {
					t.Errorf("makeTransport().RegistriesCertificates certificate appended to system certificates while added for other registry")
				}
			},
		},
		{
			name: "RegistriesClientCertificates set for registry",
			opts: config.RegistryOptions{RegistriesClientCertificates: map[string]string{registryName: "/path/to/client/certificate.cert,/path/to/client/key.key"}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if len(config.Certificates) != 1 {
					t.Errorf("makeTransport().RegistriesClientCertificates not loaded for desired registry")
				}
			},
		},
		{
			name: "RegistriesClientCertificates set for another registry",
			opts: config.RegistryOptions{RegistriesClientCertificates: map[string]string{fmt.Sprintf("other.%s", registryName): "/path/to/client/certificate.cert,/path/to/key.key,/path/to/extra.crt"}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if len(config.Certificates) != 0 {
					t.Errorf("makeTransport().RegistriesClientCertificates certificate loaded for other registry")
				}
			},
		},
		{
			name: "RegistriesClientCertificates incorrect cert format",
			opts: config.RegistryOptions{RegistriesClientCertificates: map[string]string{registryName: "/path/to/client/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if config != nil {
					t.Errorf("makeTransport().RegistriesClientCertificates was incorrectly loaded without both client/key (config was not nil)")
				}
				expectedError := "failed to load client certificate/key 'my.registry.name=/path/to/client/certificate.cert', expected format: my.registry.name=/path/to/cert,/path/to/key"
				if err == nil {
					t.Errorf("makeTransport().RegistriesClientCertificates was incorrectly loaded without both client/key (expected error, got nil)")
				} else if err.Error() != expectedError {
					t.Errorf("makeTransport().RegistriesClientCertificates was incorrectly loaded without both client/key (expected: %s, got: %s)", expectedError, err.Error())
				}
			},
		},
		{
			name: "RegistriesClientCertificates incorrect cert format extra",
			opts: config.RegistryOptions{RegistriesClientCertificates: map[string]string{registryName: "/path/to/client/certificate.cert,/path/to/key.key,/path/to/extra.crt"}},
			check: func(config *tls.Config, pool *mockedCertPool, err error) {
				if config != nil {
					t.Errorf("makeTransport().RegistriesClientCertificates was incorrectly loaded with extra paths in comma split (config was not nil)")
				}
				expectedError := "failed to load client certificate/key 'my.registry.name=/path/to/client/certificate.cert,/path/to/key.key,/path/to/extra.crt', expected format: my.registry.name=/path/to/cert,/path/to/key"
				if err == nil {
					t.Errorf("makeTransport().RegistriesClientCertificates was incorrectly loaded loaded with extra paths in comma split (expected error, got nil)")
				} else if err.Error() != expectedError {
					t.Errorf("makeTransport().RegistriesClientCertificates was incorrectly loaded loaded with extra paths in comma split (expected: %s, got: %s)", expectedError, err.Error())
				}
			},
		},
	}
	savedSystemCertLoader := systemCertLoader
	savedSystemKeyPairLoader := systemKeyPairLoader
	defer func() {
		systemCertLoader = savedSystemCertLoader
		systemKeyPairLoader = savedSystemKeyPairLoader
	}()
	for _, tt := range tests {
		var certificatesPath []string
		certPool := &mockedCertPool{
			certificatesPath: certificatesPath,
		}
		systemCertLoader = certPool
		systemKeyPairLoader = &mockedKeyPairLoader{}
		t.Run(tt.name, func(t *testing.T) {
			tr, err := MakeTransport(tt.opts, registryName)
			var tlsConfig *tls.Config
			if err == nil {
				tlsConfig = tr.(*http.Transport).TLSClientConfig
			}
			tt.check(tlsConfig, certPool, err)
		})

	}
}
