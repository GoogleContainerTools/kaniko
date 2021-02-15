module github.com/GoogleContainerTools/kaniko

go 1.14

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/containerd/containerd v1.4.0-0.20191014053712-acdcf13d5eaf => github.com/containerd/containerd v0.0.0-20191014053712-acdcf13d5eaf
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c => github.com/docker/docker v0.0.0-20190319215453-e7b5f7dbe98c
	github.com/tonistiigi/fsutil v0.0.0-20190819224149-3d2716dd0a4d => github.com/tonistiigi/fsutil v0.0.0-20191018213012-0f039a052ca1
)

require (
	cloud.google.com/go/storage v1.8.0
	github.com/Azure/azure-pipeline-go v0.2.2 // indirect
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/aws/aws-sdk-go v1.31.6
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916 // indirect
	github.com/docker/swarmkit v1.12.1-0.20180726190244-7567d47988d8 // indirect
	github.com/genuinetools/bpfd v0.0.2-0.20190525234658-c12d8cd9aac8
	github.com/go-git/go-billy/v5 v5.0.0
	github.com/go-git/go-git/v5 v5.1.0
	github.com/golang/mock v1.4.3
	github.com/google/go-cmp v0.5.2
	github.com/google/go-containerregistry v0.4.1-0.20210128200529-19c2b639fab1
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210208222243-cbafe638a7a9
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/martian v2.1.1-0.20190517191504-25dcb96d9e51+incompatible // indirect
	github.com/google/slowjam v1.0.0
	github.com/hashicorp/go-memdb v0.0.0-20180223233045-1289e7fffe71 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/karrick/godirwalk v1.16.1
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-shellwords v1.0.10 // indirect
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
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	google.golang.org/api v0.25.0 // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
	k8s.io/code-generator v0.20.1 // indirect
)
