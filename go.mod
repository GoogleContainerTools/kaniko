module github.com/GoogleContainerTools/kaniko

go 1.14

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible
	github.com/containerd/containerd v1.4.0-0.20191014053712-acdcf13d5eaf => github.com/containerd/containerd v0.0.0-20191014053712-acdcf13d5eaf
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c => github.com/docker/docker v0.0.0-20190319215453-e7b5f7dbe98c
	github.com/tonistiigi/fsutil v0.0.0-20190819224149-3d2716dd0a4d => github.com/tonistiigi/fsutil v0.0.0-20191018213012-0f039a052ca1
)

require (
	cloud.google.com/go v0.57.0 // indirect
	cloud.google.com/go/storage v1.8.0
	github.com/Azure/azure-pipeline-go v0.2.2 // indirect
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/aws/aws-sdk-go v1.31.6
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916 // indirect
	github.com/docker/swarmkit v1.12.1-0.20180726190244-7567d47988d8 // indirect
	github.com/genuinetools/amicontained v0.4.3
	github.com/go-git/go-git/v5 v5.1.0
	github.com/golang/mock v1.4.3
	github.com/google/go-cmp v0.4.1
	github.com/google/go-containerregistry v0.1.2-0.20200804170047-b0d31a182cf0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/hashicorp/go-memdb v0.0.0-20180223233045-1289e7fffe71 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/karrick/godirwalk v1.7.7
	github.com/minio/highwayhash v1.0.0
	github.com/moby/buildkit v0.0.0-20191111154543-00bfbab0390c
	github.com/opencontainers/runtime-spec v1.0.1 // indirect
	github.com/opencontainers/selinux v1.0.0-rc1 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/tonistiigi/fsutil v0.0.0-20191018213012-0f039a052ca1 // indirect
	github.com/vbatts/tar-split v0.10.2 // indirect
	golang.org/x/net v0.0.0-20200520182314-0ba52f642ac2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
)
