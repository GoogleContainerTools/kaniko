module github.com/vdemeester/k8s-pkg-credentialprovider

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v38.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.3
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/aws/aws-sdk-go v1.28.2
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/component-base v0.18.8
	k8s.io/klog v1.0.0
	k8s.io/legacy-cloud-providers v0.18.8
	sigs.k8s.io/yaml v1.2.0
)
