module github.com/GoogleContainerTools/kaniko

go 1.13

replace (
	github.com/containerd/containerd v1.4.0-0.20191014053712-acdcf13d5eaf => github.com/containerd/containerd v0.0.0-20191014053712-acdcf13d5eaf
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c => github.com/docker/docker v0.0.0-20190319215453-e7b5f7dbe98c
	github.com/tonistiigi/fsutil v0.0.0-20190819224149-3d2716dd0a4d => github.com/tonistiigi/fsutil v0.0.0-20191018213012-0f039a052ca1
)

require (
	cloud.google.com/go v0.26.0
	github.com/Azure/azure-pipeline-go v0.2.2 // indirect
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/aws/aws-sdk-go v1.25.19
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916 // indirect
	github.com/docker/swarmkit v1.12.1-0.20180726190244-7567d47988d8 // indirect
	github.com/emirpasic/gods v1.9.0 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/genuinetools/amicontained v0.4.3
	github.com/gliderlabs/ssh v0.2.2 // indirect
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.3.0
	github.com/google/go-containerregistry v0.0.0-20191218175032-34fb8ff33bed
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/martian v2.1.0+incompatible // indirect
	github.com/google/uuid v1.0.0 // indirect
	github.com/googleapis/gax-go v2.0.0+incompatible // indirect
	github.com/hashicorp/go-memdb v0.0.0-20180223233045-1289e7fffe71 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/karrick/godirwalk v1.7.7
	github.com/kevinburke/ssh_config v0.0.0-20180830205328-81db2a75821e // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190805055040-f9202b1cfdeb // indirect
	github.com/mattn/go-shellwords v1.0.3 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/minio/highwayhash v1.0.0
	github.com/moby/buildkit v0.0.0-20191111154543-00bfbab0390c
	github.com/opencontainers/runtime-spec v1.0.1 // indirect
	github.com/opencontainers/selinux v1.0.0-rc1 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.0-pre1.0.20180210140205-a40133b69fbd // indirect
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910 // indirect
	github.com/prometheus/common v0.0.0-20180518154759-7600349dcfe1 // indirect
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/src-d/gcfg v1.3.0 // indirect
	github.com/tonistiigi/fsutil v0.0.0-20191018213012-0f039a052ca1 // indirect
	github.com/vbatts/tar-split v0.10.2 // indirect
	github.com/xanzy/ssh-agent v0.2.0 // indirect
	go.opencensus.io v0.14.0 // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/api v0.0.0-20180730000901-31ca0e01cd79 // indirect
	gopkg.in/src-d/go-billy.v4 v4.2.0 // indirect
	gopkg.in/src-d/go-git-fixtures.v3 v3.5.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.6.0
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
