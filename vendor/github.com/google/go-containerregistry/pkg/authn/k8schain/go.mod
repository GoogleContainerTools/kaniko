module github.com/google/go-containerregistry/pkg/authn/k8schain

go 1.14

require (
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/go-containerregistry v0.4.1-0.20210128200529-19c2b639fab1
	github.com/google/uuid v1.1.2 // indirect
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/vdemeester/k8s-pkg-credentialprovider v1.19.7
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777 // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/klog/v2 v2.5.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210113233702-8566a335510f // indirect
)

replace github.com/google/go-containerregistry => ../../..
