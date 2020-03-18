module github.com/vdemeester/k8s-pkg-credentialprovider

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v38.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.3
	github.com/Azure/go-autorest/autorest/adal v0.8.1
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/aws/aws-sdk-go v1.27.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.4
	k8s.io/component-base v0.17.4
	k8s.io/klog v1.0.0
	k8s.io/legacy-cloud-providers v0.17.4
	sigs.k8s.io/yaml v1.1.0
)
