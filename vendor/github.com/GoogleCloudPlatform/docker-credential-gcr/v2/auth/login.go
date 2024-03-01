// Copyright 2016 Google, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package auth implements the logic required to authenticate the user and
generate access tokens for use with GCR.
*/
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/config"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/oauth2"
)

const redirectURIAuthCodeInTitleBar = "urn:ietf:wg:oauth:2.0:oob"

// GCRLoginAgent implements the OAuth2 login dance, generating an Oauth2 access_token
// for the user. If AllowBrowser is set to true, the agent will attempt to
// obtain an authorization_code automatically by executing OpenBrowser and
// reading the redirect performed after a successful login. Otherwise, it will
// attempt to use In and Out to direct the user to the login portal and receive
// the authorization_code in response.
type GCRLoginAgent struct {
	// Read input from here; if nil, uses os.Stdin.
	In io.Reader

	// Write output to here; if nil, uses os.Stdout.
	Out io.Writer

	// Open the browser for the given url.  If nil, uses webbrowser.Open.
	OpenBrowser func(url string) error
}

// populate missing fields as described in the struct definition comments
func (a *GCRLoginAgent) init() {
	if a.In == nil {
		a.In = os.Stdin
	}
	if a.Out == nil {
		a.Out = os.Stdout
	}
	if a.OpenBrowser == nil {
		a.OpenBrowser = webbrowser.Open
	}
}

// PerformLogin performs the auth dance necessary to obtain an
// authorization_code from the user and exchange it for an Oauth2 access_token.
func (a *GCRLoginAgent) PerformLogin() (*oauth2.Token, error) {
	a.init()
	conf := &oauth2.Config{
		ClientID:     config.GCRCredHelperClientID,
		ClientSecret: config.GCRCredHelperClientNotSoSecret,
		Scopes:       config.GCRScopes,
		Endpoint:     config.GCROAuth2Endpoint,
	}

	verifier, challenge, method, err := codeChallengeParams()
	state, err := makeRandString(16)
	if err != nil {
		return nil, fmt.Errorf("Unable to build random string: %v", err)
	}
	authCodeOpts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", method),
	}

	// Browser based auth is the only mechanism supported now.
	// Attempt to receive the authorization code via redirect URL
	ln, port, err := getListener()
	if err != nil {
		return nil, fmt.Errorf("Unable to open local listener: %v", err)
	}
	defer ln.Close()

	// open a web browser and listen on the redirect URL port
	conf.RedirectURL = fmt.Sprintf("http://localhost:%d", port)
	url := conf.AuthCodeURL(state, authCodeOpts...)
	err = a.OpenBrowser(url)
	if err != nil {
		return nil, fmt.Errorf("Unable to open browser: %v", err)
	}

	code, err := handleCodeResponse(ln, state)
	if err != nil {
		return nil, fmt.Errorf("Response was invalid: %v", err)
	}

	return conf.Exchange(
		config.OAuthHTTPContext,
		code,
		oauth2.SetAuthURLParam("code_verifier", verifier))
}

func (a *GCRLoginAgent) codeViaPrompt(conf *oauth2.Config, authCodeOpts []oauth2.AuthCodeOption) (string, error) {
	// Direct the user to our login portal
	conf.RedirectURL = redirectURIAuthCodeInTitleBar
	url := conf.AuthCodeURL("state", authCodeOpts...)
	fmt.Fprintln(a.Out, "Please visit the following URL and complete the authorization dialog:")
	fmt.Fprintf(a.Out, "%v\n", url)

	// Receive the authorization_code in response
	fmt.Fprintln(a.Out, "Authorization code:")
	var code string
	if _, err := fmt.Fscan(a.In, &code); err != nil {
		return "", err
	}

	return code, nil
}

func getListener() (net.Listener, int, error) {
	laddr := net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0} // port: 0 == find free port
	ln, err := net.ListenTCP("tcp4", &laddr)
	if err != nil {
		return nil, 0, err
	}
	return ln, ln.Addr().(*net.TCPAddr).Port, nil
}

func handleCodeResponse(ln net.Listener, stateCheck string) (string, error) {
	conn, err := ln.Accept()
	if err != nil {
		return "", err
	}

	srvConn := httputil.NewServerConn(conn, nil)
	defer srvConn.Close()

	req, err := srvConn.Read()
	if err != nil {
		return "", err
	}

	code := req.URL.Query().Get("code")
	state := req.URL.Query().Get("state")

	resp := &http.Response{
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Close:         true,
		ContentLength: -1, // designates unknown length
	}
	defer srvConn.Write(req, resp)

	// If the code couldn't be obtained, inform the user via the browser and
	// return an error.
	// TODO i18n?
	if code == "" {
		err := fmt.Errorf("Code not present in response: %s", req.URL.String())
		resp.Body = getResponseBody("ERROR: Authentication code not present in response.")
		return "", err
	}

	if state != stateCheck {
		err := fmt.Errorf("Invalid State")
		resp.StatusCode = 400
		resp.Body = getResponseBody("ERROR: State parameter is invalid.")
		return "", err
	}

	resp.Body = getResponseBody("Success! You may now close your browser.")
	return code, nil
}

// turn a string into an io.ReadCloser as required by an http.Response
func getResponseBody(body string) io.ReadCloser {
	reader := strings.NewReader(body)
	return ioutil.NopCloser(reader)
}

// generates the values used in "Proof Key for Code Exchange by OAuth Public Clients"
// https://tools.ietf.org/html/rfc7636
// https://developers.google.com/identity/protocols/OAuth2InstalledApp#step1-code-verifier
func codeChallengeParams() (verifier, challenge, method string, err error) {
	// A `code_verifier` is a high-entropy cryptographic random string using the unreserved characters
	// [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
	// with a minimum length of 43 characters and a maximum length of 128 characters.
	verifier, err = makeRandString(32)
	if err != nil {
		return "", "", "", err
	}

	// https://tools.ietf.org/html/rfc7636#section-4.2
	// If the client is capable of using "S256", it MUST use "S256":
	// code_challenge = BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
	sha := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sha[:])

	return verifier, challenge, "S256", nil
}

func makeRandString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
