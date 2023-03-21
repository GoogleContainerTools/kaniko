module github.com/GoogleContainerTools/kaniko

go 1.17

// These match the docker/docker's dependencies configured in:
// https://github.com/moby/moby/blob/v20.10.12/vendor.conf
replace (
	github.com/moby/buildkit => github.com/moby/buildkit v0.8.3
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc92
	github.com/tonistiigi/fsutil => github.com/tonistiigi/fsutil v0.0.0-20201103201449-0834f99b7b85
)

require (
	cloud.google.com/go/storage v1.29.0
	github.com/Azure/azure-storage-blob-go v0.14.0
	github.com/aws/aws-sdk-go v1.44.24
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20220228164355-396b2034c795
	github.com/chrismellard/docker-credential-acr-env v0.0.0-20220119192733-fe33c00cee21
	github.com/containerd/cgroups v1.0.3 // indirect
	github.com/docker/docker v20.10.20+incompatible
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.9
	github.com/google/go-containerregistry v0.13.1-0.20230201183932-824efc7772b0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/slowjam v1.0.0
	github.com/karrick/godirwalk v1.16.1
	github.com/minio/highwayhash v1.0.2
	github.com/moby/buildkit v0.9.3
	github.com/otiai10/copy v1.7.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/afero v1.8.2
	github.com/spf13/cobra v1.6.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.1.0
	golang.org/x/oauth2 v0.1.0
	golang.org/x/sync v0.1.0
)

require github.com/containerd/containerd v1.6.2

require (
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.1 // indirect
	golang.org/x/mod v0.6.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
)

require (
	cloud.google.com/go v0.107.0 // indirect
	cloud.google.com/go/compute v1.14.0 // indirect
	cloud.google.com/go/iam v0.8.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-sdk-for-go v61.3.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.24 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.18 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.11 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.5 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Microsoft/hcsshim v0.9.2 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20220113124808-70ae35bab23f // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/aws/aws-sdk-go-v2 v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.9.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.3.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.10.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.15.0 // indirect
	github.com/aws/smithy-go v1.11.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cilium/ebpf v0.8.0 // indirect
	github.com/containerd/continuity v0.2.2 // indirect
	github.com/containerd/fifo v1.0.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.1 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/coreos/etcd v3.3.27+incompatible // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/cli v20.10.20+incompatible // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/docker/swarmkit v1.12.1-0.20180726190244-7567d47988d8 // indirect
	github.com/ePirat/docker-credential-gitlabci v1.0.0
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.7.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kevinburke/ssh_config v1.1.0 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/mattn/go-ieproxy v0.0.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/sys/mount v0.3.0 // indirect
	github.com/moby/sys/mountinfo v0.5.0 // indirect
	github.com/moby/sys/symlink v0.2.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/opencontainers/runc v1.1.0 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417 // indirect
	github.com/opencontainers/selinux v1.10.0 // indirect
	github.com/prometheus/client_golang v1.12.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/tonistiigi/fsutil v0.0.0-20220115021204-b19f7f9cb274 // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/xanzy/ssh-agent v0.3.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/sys v0.1.0
	golang.org/x/text v0.5.0 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.106.0
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.51.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
