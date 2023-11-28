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
	"os"
	"strings"

	"net/http"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/sirupsen/logrus"
)

type CertPool interface {
	value() *x509.CertPool
	append(path string) error
}

type X509CertPool struct {
	inner x509.CertPool
}

func (p *X509CertPool) value() *x509.CertPool {
	return &p.inner
}

func (p *X509CertPool) append(path string) error {
	pem, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	p.inner.AppendCertsFromPEM(pem)
	return nil
}

var systemCertLoader CertPool

type KeyPairLoader interface {
	load(string, string) (tls.Certificate, error)
}

type X509KeyPairLoader struct {
}

func (p *X509KeyPairLoader) load(certFile, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

var systemKeyPairLoader KeyPairLoader

func init() {
	systemCertPool, err := x509.SystemCertPool()
	if err != nil {
		logrus.Warn("Failed to load system cert pool. Loading empty one instead.")
		systemCertPool = x509.NewCertPool()
	}
	systemCertLoader = &X509CertPool{
		inner: *systemCertPool,
	}

	systemKeyPairLoader = &X509KeyPairLoader{}
}

func MakeTransport(opts config.RegistryOptions, registryName string) (http.RoundTripper, error) {
	// Create a transport to set our user-agent.
	var tr http.RoundTripper = http.DefaultTransport.(*http.Transport).Clone()
	if opts.SkipTLSVerify || opts.SkipTLSVerifyRegistries.Contains(registryName) {
		tr.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else if certificatePath := opts.RegistriesCertificates[registryName]; certificatePath != "" {
		if err := systemCertLoader.append(certificatePath); err != nil {
			return nil, fmt.Errorf("failed to load certificate %s for %s: %w", certificatePath, registryName, err)
		}
		tr.(*http.Transport).TLSClientConfig = &tls.Config{
			RootCAs: systemCertLoader.value(),
		}
	}

	if clientCertificatePath := opts.RegistriesClientCertificates[registryName]; clientCertificatePath != "" {
		certFiles := strings.Split(clientCertificatePath, ",")
		if len(certFiles) != 2 {
			return nil, fmt.Errorf("failed to load client certificate/key '%s=%s', expected format: %s=/path/to/cert,/path/to/key", registryName, clientCertificatePath, registryName)
		}
		cert, err := systemKeyPairLoader.load(certFiles[0], certFiles[1])
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate/key '%s' for %s: %w", clientCertificatePath, registryName, err)
		}
		tr.(*http.Transport).TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	return tr, nil
}
