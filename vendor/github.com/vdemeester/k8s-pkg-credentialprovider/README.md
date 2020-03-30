# k8s.io/kubernetes/pkg/credentialprovider temporary fork

This is a temporary fork of the package
`k8s.io/kubernetes/pkg/credentialprovider` to be able to use it
without having to depend on `k8s.io/kubernetes`, dependency that
brings hell with it, **especially** with go modules.

See [kubernetes/enhancements#1406](https://github.com/kubernetes/enhancements/pull/1406) for progress on the matter.
