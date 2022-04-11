/* Copyright (c) 2022 Marvin Scholz <epirat07 at gmail dot com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package credhelper

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
)

type GitLabCredentialsHelper struct{}

var ErrUnsupportedOperation = errors.New("unsupported operation")

func NewGitLabCredentialsHelper() credentials.Helper {
	return &GitLabCredentialsHelper{}
}

func parseRegistryURL(urlString string) (*url.URL, error) {
	// If no scheme is given, use https as it is the default
	// for docker registries
	if !(strings.HasPrefix(urlString, "https://") ||
		strings.HasPrefix(urlString, "http://")) {
		urlString = "https://" + urlString
	}
	return url.Parse(urlString)
}

func matchRegistryURL(serverURLString string) error {
	// Read registry URL from predefined CI environment variable
	// https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
	envURLString, urlFound := os.LookupEnv("CI_REGISTRY")
	if !urlFound {
		return errors.New("no CI_REGISTRY env var set")
	}
	envURL, err := parseRegistryURL(envURLString)
	if err != nil {
		return fmt.Errorf("failed to parse CI_REGISTRY URL: %w", err)
	}
	serverURL, err := parseRegistryURL(serverURLString)
	if err != nil {
		return fmt.Errorf("failed to parse registry URL: %w", err)
	}
	if serverURL.Hostname() == "" || envURL.Hostname() == "" {
		return errors.New("failed getting hosts for matching")
	}
	if serverURL.Scheme != envURL.Scheme {
		return fmt.Errorf("protocol for '%s' does not match CI_REGISTRY host '%s'",
			serverURLString,
			envURLString)
	}
	if serverURL.Hostname() != envURL.Hostname() {
		return fmt.Errorf("host '%s' does not match CI_REGISTRY host '%s'",
			serverURL.Hostname(),
			envURL.Hostname())
	}
	return nil
}

func (credHelper GitLabCredentialsHelper) Add(c *credentials.Credentials) error {
	return ErrUnsupportedOperation
}

func (credHelper GitLabCredentialsHelper) Delete(serverURL string) error {
	return ErrUnsupportedOperation
}

func (credHelper GitLabCredentialsHelper) Get(serverURL string) (string, string, error) {
	if err := matchRegistryURL(serverURL); err != nil {
		return "", "", fmt.Errorf("server URL does not match CI_REGISTRY URL: %w", err)
	}
	user, found := os.LookupEnv("CI_REGISTRY_USER")
	if !found {
		return "", "", errors.New("no CI_REGISTRY_USER env var set")
	}
	pass, found := os.LookupEnv("CI_REGISTRY_PASSWORD")
	if !found {
		return "", "", errors.New("no CI_REGISTRY_PASSWORD env var set")
	}
	return user, pass, nil
}

func (credHelper GitLabCredentialsHelper) List() (map[string]string, error) {
	return nil, ErrUnsupportedOperation
}
