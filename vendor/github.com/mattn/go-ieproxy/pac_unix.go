// +build !windows,!darwin

package ieproxy

func (psc *ProxyScriptConf) findProxyForURL(URL string) string {
	return ""
}
