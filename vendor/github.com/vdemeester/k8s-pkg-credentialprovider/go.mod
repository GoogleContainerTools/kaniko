module github.com/vdemeester/k8s-pkg-credentialprovider

go 1.15

require (
	github.com/Azure/azure-sdk-for-go v43.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/aws/aws-sdk-go v1.28.2
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/component-base v0.19.7
	k8s.io/klog/v2 v2.2.0
	k8s.io/legacy-cloud-providers v0.19.7
	sigs.k8s.io/yaml v1.2.0
)
