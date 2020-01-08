// Package autoload automatically calls OverrideEnvWithStaticProxy,
// which writes new values to the `http_proxy`, `https_proxy` and `no_proxy` environment variables.
// The values are taken from the Windows Regedit
// import _ "github.com/mattn/go-ieproxy/autoload"
package autoload

import ieproxy "github.com/mattn/go-ieproxy"

func init() {
	ieproxy.OverrideEnvWithStaticProxy()
}
