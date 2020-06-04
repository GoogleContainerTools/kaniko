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

func Test_makeTransport(t *testing.T) {
	registryName := "my.registry.name"

	tests := []struct {
		name  string
		opts  *config.KanikoOptions
		check func(*tls.Config, *mockedCertPool)
	}{
		{
			name: "SkipTLSVerify set",
			opts: &config.KanikoOptions{SkipTLSVerify: true},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if !config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify not set while SkipTLSVerify set")
				}
			},
		},
		{
			name: "SkipTLSVerifyRegistries set with expected registry",
			opts: &config.KanikoOptions{SkipTLSVerifyRegistries: []string{registryName}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if !config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify not set while SkipTLSVerifyRegistries set with registry name")
				}
			},
		},
		{
			name: "SkipTLSVerifyRegistries set with other registry",
			opts: &config.KanikoOptions{SkipTLSVerifyRegistries: []string{fmt.Sprintf("other.%s", registryName)}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify set while SkipTLSVerifyRegistries not set with registry name")
				}
			},
		},
		{
			name: "RegistriesCertificates set for registry",
			opts: &config.KanikoOptions{RegistriesCertificates: map[string]string{registryName: "/path/to/the/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if len(pool.certificatesPath) != 1 || pool.certificatesPath[0] != "/path/to/the/certificate.cert" {
					t.Errorf("makeTransport().RegistriesCertificates certificate not appended to system certificates")
				}
			},
		},
		{
			name: "RegistriesCertificates set for another registry",
			opts: &config.KanikoOptions{RegistriesCertificates: map[string]string{fmt.Sprintf("other.%s=", registryName): "/path/to/the/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if len(pool.certificatesPath) != 0 {
					t.Errorf("makeTransport().RegistriesCertificates certificate appended to system certificates while added for other registry")
				}
			},
		},
	}
	savedSystemCertLoader := systemCertLoader
	defer func() { systemCertLoader = savedSystemCertLoader }()
	for _, tt := range tests {
		var certificatesPath []string
		certPool := &mockedCertPool{
			certificatesPath: certificatesPath,
		}
		systemCertLoader = certPool
		t.Run(tt.name, func(t *testing.T) {
			tr := MakeTransport(tt.opts, registryName)
			tt.check(tr.(*http.Transport).TLSClientConfig, certPool)
		})

	}
}
