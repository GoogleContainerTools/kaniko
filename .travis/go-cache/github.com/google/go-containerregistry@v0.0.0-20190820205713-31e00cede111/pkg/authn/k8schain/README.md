# `k8schain`

This is an implementation of the [github.com/google/go-containerregistry](
https://github.com/google/go-containerregistry) library's [`authn.Keychain`](
https://godoc.org/github.com/google/go-containerregistry/authn#Keychain)
interface based on the authentication semantics used by the Kubelet when
performing the pull of a Pod's images.

## Usage

### Creating a keychain

A `k8schain` keychain can be built via one of:

```go
// client is a kubernetes.Interface
kc, err := k8schain.New(client, k8schain.Options{})
...

// This method is suitable for use by controllers or other in-cluster processes.
kc, err := k8schain.NewInCluster(k8schain.Options{})
...
```

### Using the keychain

The `k8schain` keychain can be used directly as an `authn.Keychain`, e.g.

```go
	auth, err := kc.Resolve(registry)
	if err != nil {
		...
	}
```

Or, it can be used to override the default keychain used by this process,
which by default follows Docker's keychain semantics:

```go
func init() {
	// Override the default keychain used by this process to follow the
	// Kubelet's keychain semantics.
	authn.DefaultKeychain = kc
}
```
