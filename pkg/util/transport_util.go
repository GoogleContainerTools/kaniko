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

	"io/ioutil"
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
	pem, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	p.inner.AppendCertsFromPEM(pem)
	return nil
}

var systemCertLoader CertPool

func init() {
	systemCertPool, err := x509.SystemCertPool()
	if err != nil {
		logrus.Warn("Failed to load system cert pool. Loading empty one instead.")
		systemCertPool = x509.NewCertPool()
	}
	systemCertLoader = &X509CertPool{
		inner: *systemCertPool,
	}
}

func MakeTransport(opts *config.KanikoOptions, registryName string) http.RoundTripper {
	// Create a transport to set our user-agent.
	var tr http.RoundTripper = http.DefaultTransport.(*http.Transport).Clone()
	if opts.SkipTLSVerify || opts.SkipTLSVerifyRegistries.Contains(registryName) {
		tr.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else if certificatePath := opts.RegistriesCertificates[registryName]; certificatePath != "" {
		if err := systemCertLoader.append(certificatePath); err != nil {
			logrus.WithError(err).Warnf("Failed to load certificate %s for %s\n", certificatePath, registryName)
		} else {
			tr.(*http.Transport).TLSClientConfig = &tls.Config{
				RootCAs: systemCertLoader.value(),
			}
		}
	}
	return tr
}
