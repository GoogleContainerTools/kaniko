# v1.23.2 Release 2024-07-09
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.23.2
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.23.2-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.23.2-slim
```

* deps: bump github.com/moby/buildkit and github.com/docker/docker [#3242](https://github.com/GoogleContainerTools/kaniko/pull/3242)
* chore(deps): bump docker/build-push-action from 6.1.0 to 6.3.0 [#3236](https://github.com/GoogleContainerTools/kaniko/pull/3236)
* chore(deps): bump docker/setup-qemu-action from 3.0.0 to 3.1.0 [#3235](https://github.com/GoogleContainerTools/kaniko/pull/3235)
* chore(deps): bump docker/setup-buildx-action from 3.3.0 to 3.4.0 [#3237](https://github.com/GoogleContainerTools/kaniko/pull/3237)
* chore(deps): bump google.golang.org/api from 0.185.0 to 0.187.0 [#3238](https://github.com/GoogleContainerTools/kaniko/pull/3238)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.17.1 to 1.17.5 [#3239](https://github.com/GoogleContainerTools/kaniko/pull/3239)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.24 to 1.17.1 [#3220](https://github.com/GoogleContainerTools/kaniko/pull/3220)
* chore(deps): bump docker/build-push-action from 6.0.0 to 6.1.0 [#3218](https://github.com/GoogleContainerTools/kaniko/pull/3218)
* chore(deps): bump google.golang.org/api from 0.183.0 to 0.185.0 [#3219](https://github.com/GoogleContainerTools/kaniko/pull/3219)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.55.1 to 1.56.1 [#3221](https://github.com/GoogleContainerTools/kaniko/pull/3221)
* chore(deps): bump docker/build-push-action from 5.3.0 to 6.0.0 [#3212](https://github.com/GoogleContainerTools/kaniko/pull/3212)
* chore(deps): bump cloud.google.com/go/storage from 1.41.0 to 1.42.0 [#3204](https://github.com/GoogleContainerTools/kaniko/pull/3204)
* chore(deps): bump github.com/spf13/cobra from 1.8.0 to 1.8.1 [#3205](https://github.com/GoogleContainerTools/kaniko/pull/3205)
* chore(deps): bump github.com/google/go-containerregistry from 0.19.1 to 0.19.2 [#3206](https://github.com/GoogleContainerTools/kaniko/pull/3206)
* chore(deps): bump imjasonh/setup-crane from 0.3 to 0.4 [#3210](https://github.com/GoogleContainerTools/kaniko/pull/3210)
* chore(deps): bump golang.org/x/net from 0.25.0 to 0.26.0 [#3190](https://github.com/GoogleContainerTools/kaniko/pull/3190)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.21 to 1.16.24 [#3191](https://github.com/GoogleContainerTools/kaniko/pull/3191)
* chore(deps): bump google.golang.org/api from 0.182.0 to 0.183.0 [#3192](https://github.com/GoogleContainerTools/kaniko/pull/3192)
* chore(deps): bump github.com/containerd/containerd from 1.7.17 to 1.7.18 [#3193](https://github.com/GoogleContainerTools/kaniko/pull/3193)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.27.0 to 1.27.2 [#3194](https://github.com/GoogleContainerTools/kaniko/pull/3194)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]


# v1.23.1 Release 2024-06-07
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.23.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.23.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.23.1-slim
```

* Enable pushing cache with --no-push [#3181](https://github.com/GoogleContainerTools/kaniko/pull/3181)
* docs: document --no-push-cache flag in README.md [#3188](https://github.com/GoogleContainerTools/kaniko/pull/3188)
* chore(deps): bump google.golang.org/api from 0.181.0 to 0.182.0 [#3187](https://github.com/GoogleContainerTools/kaniko/pull/3187)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.17 to 1.16.21 [#3179](https://github.com/GoogleContainerTools/kaniko/pull/3179)
* chore(deps): bump google.golang.org/api from 0.180.0 to 0.181.0 [#3170](https://github.com/GoogleContainerTools/kaniko/pull/3170)
* chore(deps): bump google-github-actions/auth from 2.1.2 to 2.1.3 [#3168](https://github.com/GoogleContainerTools/kaniko/pull/3168)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.53.2 to 1.54.2 [#3169](https://github.com/GoogleContainerTools/kaniko/pull/3169)
* chore(deps): bump cloud.google.com/go/storage from 1.40.0 to 1.41.0 [#3171](https://github.com/GoogleContainerTools/kaniko/pull/3171)
* chore(deps): bump github.com/containerd/containerd from 1.7.16 to 1.7.17 [#3172](https://github.com/GoogleContainerTools/kaniko/pull/3172)
* chore(deps): bump github.com/docker/docker from 26.1.2+incompatible to 26.1.3+incompatible [#3173](https://github.com/GoogleContainerTools/kaniko/pull/3173)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- Leo Palmer Sunmo


# v1.23.0 Release 2024-05-14
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.23.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.23.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.23.0-slim
```

* give warn instead of error when wildcard not match any files [#3127](https://github.com/GoogleContainerTools/kaniko/pull/3127)
* warmer validate and copy registry mirror to registry map [#3140](https://github.com/GoogleContainerTools/kaniko/pull/3140)
* docs: update docs on mirrors and registry map. [#3153](https://github.com/GoogleContainerTools/kaniko/pull/3153)
* Fix: Make `--registry-map` compatible with namespaced images [#3138](https://github.com/GoogleContainerTools/kaniko/pull/3138)
* "Fixes #2752" [#3132](https://github.com/GoogleContainerTools/kaniko/pull/3132)
* chore(deps): bump github.com/docker/docker from 26.1.1+incompatible to 26.1.2+incompatible [#3161](https://github.com/GoogleContainerTools/kaniko/pull/3161)
* chore(deps): bump google.golang.org/api from 0.177.0 to 0.180.0 [#3160](https://github.com/GoogleContainerTools/kaniko/pull/3160)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.15 to 1.16.17 [#3158](https://github.com/GoogleContainerTools/kaniko/pull/3158)
* chore(deps): bump github.com/docker/docker from 26.1.0+incompatible to 26.1.1+incompatible [#3149](https://github.com/GoogleContainerTools/kaniko/pull/3149)
* chore(deps): bump actions/setup-go from 5.0.0 to 5.0.1 [#3152](https://github.com/GoogleContainerTools/kaniko/pull/3152)
* chore(deps): bump google.golang.org/api from 0.175.0 to 0.177.0 [#3151](https://github.com/GoogleContainerTools/kaniko/pull/3151)
* chore(deps): bump golang.org/x/oauth2 from 0.19.0 to 0.20.0 [#3150](https://github.com/GoogleContainerTools/kaniko/pull/3150)
* chore(deps): bump github.com/moby/buildkit from 0.13.1 to 0.13.2 [#3145](https://github.com/GoogleContainerTools/kaniko/pull/3145)
* chore(deps): bump github.com/containerd/containerd from 1.7.15 to 1.7.16 [#3144](https://github.com/GoogleContainerTools/kaniko/pull/3144)
* chore: bump cred helper libraries [#3133](https://github.com/GoogleContainerTools/kaniko/pull/3133)
* Added --chmod for ADD and COPY commands. Fixes #2850 and #1751 [#3119](https://github.com/GoogleContainerTools/kaniko/pull/3119)
* chore(deps): bump github.com/google/slowjam from 1.1.0 to 1.1.1 [#3129](https://github.com/GoogleContainerTools/kaniko/pull/3129)
* chore(deps): bump google.golang.org/api from 0.172.0 to 0.175.0 [#3128](https://github.com/GoogleContainerTools/kaniko/pull/3128)
* fix: integration: fail on error when build with docker [#3131](https://github.com/GoogleContainerTools/kaniko/pull/3131)
* fix(doc): wiki url [#3117](https://github.com/GoogleContainerTools/kaniko/pull/3117)
* chore(deps): bump golang.org/x/net from 0.22.0 to 0.24.0 [#3113](https://github.com/GoogleContainerTools/kaniko/pull/3113)
* chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/storage/azblob from 1.3.1 to 1.3.2 [#3114](https://github.com/GoogleContainerTools/kaniko/pull/3114)
* chore(deps): bump github.com/containerd/containerd from 1.7.14 to 1.7.15 [#3112](https://github.com/GoogleContainerTools/kaniko/pull/3112)
* chore(deps): bump docker/setup-buildx-action from 3.2.0 to 3.3.0 [#3111](https://github.com/GoogleContainerTools/kaniko/pull/3111)
* chore(deps): bump github.com/docker/docker from 26.0.0+incompatible to 26.0.2+incompatible [#3121](https://github.com/GoogleContainerTools/kaniko/pull/3121)
* chore(deps): bump AdityaGarg8/remove-unwanted-software from 2 to 3 [#3110](https://github.com/GoogleContainerTools/kaniko/pull/3110)
* chore(deps): bump sigstore/cosign-installer from 3.4.0 to 3.5.0 [#3109](https://github.com/GoogleContainerTools/kaniko/pull/3109)
* chore(deps): bump golang.org/x/sys from 0.18.0 to 0.19.0 [#3103](https://github.com/GoogleContainerTools/kaniko/pull/3103)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.9 to 1.16.15 [#3104](https://github.com/GoogleContainerTools/kaniko/pull/3104)
* chore(deps): bump golang.org/x/sync from 0.6.0 to 0.7.0 [#3105](https://github.com/GoogleContainerTools/kaniko/pull/3105)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.27.7 to 1.27.11 [#3106](https://github.com/GoogleContainerTools/kaniko/pull/3106)
* chore(deps): bump golang.org/x/oauth2 from 0.18.0 to 0.19.0 [#3107](https://github.com/GoogleContainerTools/kaniko/pull/3107)
* chore(deps): bump google.golang.org/api from 0.171.0 to 0.172.0 [#3094](https://github.com/GoogleContainerTools/kaniko/pull/3094)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.52.1 to 1.53.1 [#3096](https://github.com/GoogleContainerTools/kaniko/pull/3096)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.11.0 to 5.12.0 [#3095](https://github.com/GoogleContainerTools/kaniko/pull/3095)
* chore(deps): bump github.com/moby/buildkit from 0.13.0 to 0.13.1 [#3093](https://github.com/GoogleContainerTools/kaniko/pull/3093)
* chore(deps): bump cloud.google.com/go/storage from 1.39.1 to 1.40.0 [#3097](https://github.com/GoogleContainerTools/kaniko/pull/3097)
* chore: update cred helper go libraries [#3087](https://github.com/GoogleContainerTools/kaniko/pull/3087)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- Djabx
- Marc Lallaouret
- Matthias Schneider
- Prima Adi Pradana
- Samarth08
- Verlhac Gaëtan


# v1.22.0 Release 2024-03-26
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.22.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.22.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.22.0-slim
```

* chore(deps): bump github.com/docker/docker from 25.0.4+incompatible to 26.0.0+incompatible [#3085](https://github.com/GoogleContainerTools/kaniko/pull/3085)
* chore(deps): bump google.golang.org/api from 0.167.0 to 0.171.0 [#3082](https://github.com/GoogleContainerTools/kaniko/pull/3082)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.25.3 to 1.26.0 [#3083](https://github.com/GoogleContainerTools/kaniko/pull/3083)
* chore(deps): bump github.com/containerd/containerd from 1.7.13 to 1.7.14 [#3084](https://github.com/GoogleContainerTools/kaniko/pull/3084)
* chore(deps): bump docker/build-push-action from 5.2.0 to 5.3.0 [#3070](https://github.com/GoogleContainerTools/kaniko/pull/3070)
* Fix #3032: Remove query parameters in ADD command when the destinatio… [#3053](https://github.com/GoogleContainerTools/kaniko/pull/3053)
* Kaniko/add path regmaps [possible in registry maps and/or mirror] [#3051](https://github.com/GoogleContainerTools/kaniko/pull/3051)
* chore(deps): bump docker/setup-buildx-action from 3.1.0 to 3.2.0 [#3071](https://github.com/GoogleContainerTools/kaniko/pull/3071)
* chore(deps): bump github.com/moby/buildkit from 0.12.5 to 0.13.0 [#3072](https://github.com/GoogleContainerTools/kaniko/pull/3072)
* chore(deps): bump github.com/google/go-containerregistry from 0.19.0 to 0.19.1 [#3073](https://github.com/GoogleContainerTools/kaniko/pull/3073)
* chore(deps): bump golang.org/x/oauth2 from 0.17.0 to 0.18.0 [#3074](https://github.com/GoogleContainerTools/kaniko/pull/3074)
* chore(deps): bump cloud.google.com/go/storage from 1.39.0 to 1.39.1 [#3075](https://github.com/GoogleContainerTools/kaniko/pull/3075)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.51.4 to 1.52.1 [#3076](https://github.com/GoogleContainerTools/kaniko/pull/3076)
* Fix COPY fails when multiple files are copied to path specified in ENV [#3034](https://github.com/GoogleContainerTools/kaniko/pull/3034)
* Add AWS ECR error message for tag Immutability [#3045](https://github.com/GoogleContainerTools/kaniko/pull/3045)
* chore: update google.golang.org/protobuff to resolve CVE-2024-24786 [#3068](https://github.com/GoogleContainerTools/kaniko/pull/3068)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.6 to 1.16.9 [#3058](https://github.com/GoogleContainerTools/kaniko/pull/3058)
* chore(deps): bump golang.org/x/net from 0.21.0 to 0.22.0 [#3056](https://github.com/GoogleContainerTools/kaniko/pull/3056)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.25.2 to 1.25.3 [#3057](https://github.com/GoogleContainerTools/kaniko/pull/3057)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.51.1 to 1.51.4 [#3059](https://github.com/GoogleContainerTools/kaniko/pull/3059)
* chore(deps): bump github.com/docker/docker from 25.0.3+incompatible to 25.0.4+incompatible [#3060](https://github.com/GoogleContainerTools/kaniko/pull/3060)
* chore(deps): bump docker/build-push-action from 5.1.0 to 5.2.0 [#3061](https://github.com/GoogleContainerTools/kaniko/pull/3061)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- Alessandro Bitocchi
- dependabot[bot]
- Jérémie Augustin
- Prima Adi Pradana


# v1.21.1 Release 2024-03-06
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.21.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.21.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.21.1-slim
```

* chore(deps): bump cloud.google.com/go/storage from 1.38.0 to 1.39.0 [#3040](https://github.com/GoogleContainerTools/kaniko/pull/3040)
* chore(deps): bump github.com/containerd/containerd from 1.7.6 to 1.7.13 [#3038](https://github.com/GoogleContainerTools/kaniko/pull/3038)
* test: fix test breakage caused by external dependency update [#3049](https://github.com/GoogleContainerTools/kaniko/pull/3049)
* chore(deps): bump docker/setup-buildx-action from 3.0.0 to 3.1.0 [#3037](https://github.com/GoogleContainerTools/kaniko/pull/3037)
* chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/storage/azblob from 1.3.0 to 1.3.1 [#3039](https://github.com/GoogleContainerTools/kaniko/pull/3039)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]


# v1.21.0 Release 2024-02-29
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.21.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.21.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.21.0-slim
```

* Add --push-ignore-immutable-tag-errors boolean CLI option [#2774](https://github.com/GoogleContainerTools/kaniko/pull/2774)
* docs: fix broken links and redirects [#3009](https://github.com/GoogleContainerTools/kaniko/pull/3009)
* feat: add skip tls flag for private git context [#2854](https://github.com/GoogleContainerTools/kaniko/pull/2854)
* Fix unpack tar.gz archive with ADD instruction, issue #2409 [#2991](https://github.com/GoogleContainerTools/kaniko/pull/2991)
* chore: update google github-action auth version [#3030](https://github.com/GoogleContainerTools/kaniko/pull/3030)
* refactor: remove artifact upload from nightly-vulnerabiliy-scan.yml [#3029](https://github.com/GoogleContainerTools/kaniko/pull/3029)
* feat: add nightly grype vuln scan to kaniko executor image [#2970](https://github.com/GoogleContainerTools/kaniko/pull/2970)
* chore: update docker-credential-gcr to use v2 [#3026](https://github.com/GoogleContainerTools/kaniko/pull/3026)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.16.1 to 1.16.6 [#3020](https://github.com/GoogleContainerTools/kaniko/pull/3020)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.50.0 to 1.51.1 [#3021](https://github.com/GoogleContainerTools/kaniko/pull/3021)
* chore(deps): bump google.golang.org/api from 0.165.0 to 0.167.0 [#3023](https://github.com/GoogleContainerTools/kaniko/pull/3023)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.27.0 to 1.27.4 [#3024](https://github.com/GoogleContainerTools/kaniko/pull/3024)
* chore(deps): bump google-github-actions/auth from 2.1.1 to 2.1.2 [#3025](https://github.com/GoogleContainerTools/kaniko/pull/3025)
* feat: add support for no push environment variable [#2983](https://github.com/GoogleContainerTools/kaniko/pull/2983)
* Add documentation for --chown support limitation [#3019](https://github.com/GoogleContainerTools/kaniko/pull/3019)
* chore(deps): bump github.com/Azure/azure-sdk-for-go/sdk/storage/azblob from 1.2.1 to 1.3.0 [#3013](https://github.com/GoogleContainerTools/kaniko/pull/3013)
* chore(deps): bump google.golang.org/api from 0.161.0 to 0.165.0 [#3016](https://github.com/GoogleContainerTools/kaniko/pull/3016)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.15.15 to 1.16.1 [#3014](https://github.com/GoogleContainerTools/kaniko/pull/3014)
* chore(deps): bump cloud.google.com/go/storage from 1.37.0 to 1.38.0 [#3015](https://github.com/GoogleContainerTools/kaniko/pull/3015)
* Add flag to remap registries for any registry mirror [#2935](https://github.com/GoogleContainerTools/kaniko/pull/2935)
* FIX: missing or partial support for pattern substition in variable when cache enabled [#2968](https://github.com/GoogleContainerTools/kaniko/pull/2968)
* docs: add ROADMAP.md to kaniko project [#3005](https://github.com/GoogleContainerTools/kaniko/pull/3005)
* chore: update MAINTAINERS file with up-to-date information [#3003](https://github.com/GoogleContainerTools/kaniko/pull/3003)
* chore(deps): bump golang.org/x/oauth2 from 0.16.0 to 0.17.0 [#3000](https://github.com/GoogleContainerTools/kaniko/pull/3000)
* chore(deps): bump golang.org/x/net from 0.20.0 to 0.21.0 [#2999](https://github.com/GoogleContainerTools/kaniko/pull/2999)
* chore(deps): bump golang from 1.21 to 1.22 in /deploy [#2997](https://github.com/GoogleContainerTools/kaniko/pull/2997)
* chore(deps): bump cloud.google.com/go/storage from 1.36.0 to 1.37.0 [#2998](https://github.com/GoogleContainerTools/kaniko/pull/2998)
* chore(deps): bump golang.org/x/sys from 0.16.0 to 0.17.0 [#3001](https://github.com/GoogleContainerTools/kaniko/pull/3001)
* chore(deps): bump google-github-actions/auth from 2.1.0 to 2.1.1 [#3002](https://github.com/GoogleContainerTools/kaniko/pull/3002)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- Alessandro Bitocchi
- Damien Degois
- dependabot[bot]
- JeromeJu
- Kraev Sergei
- Matheus Pimenta
- Oliver Radwell
- Sacha Smart
- schwannden


# v1.20.1 Release 2024-02-10
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.20.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.20.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.20.1-slim
```

* chore(deps): bump github.com/moby/buildkit from 0.11.6 to 0.12.5, github.com/docker/docker from 24.0.7+incompatible to 25.0.2+incompatible, and other deps [#2995](https://github.com/GoogleContainerTools/kaniko/pull/2995)
* chore(deps): bump google.golang.org/api from 0.157.0 to 0.161.0 [#2987](https://github.com/GoogleContainerTools/kaniko/pull/2987)
* chore(deps): bump github.com/google/go-containerregistry from 0.18.0 to 0.19.0 [#2988](https://github.com/GoogleContainerTools/kaniko/pull/2988)
* chore(deps): bump sigstore/cosign-installer from 3.3.0 to 3.4.0 [#2989](https://github.com/GoogleContainerTools/kaniko/pull/2989)
* chore(deps): bump github.com/opencontainers/runc from 1.1.5 to 1.1.12 [#2981](https://github.com/GoogleContainerTools/kaniko/pull/2981)
* README change only: Clarify why merging into another container is a bad idea [#2965](https://github.com/GoogleContainerTools/kaniko/pull/2965)
* chore(deps): bump google-github-actions/auth from 2.0.1 to 2.1.0 [#2972](https://github.com/GoogleContainerTools/kaniko/pull/2972)
* chore(deps): bump google-github-actions/setup-gcloud from 2.0.1 to 2.1.0 [#2973](https://github.com/GoogleContainerTools/kaniko/pull/2973)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.15.14 to 1.15.15 [#2975](https://github.com/GoogleContainerTools/kaniko/pull/2975)
* chore(deps): bump github.com/google/go-containerregistry from 0.17.0 to 0.18.0 [#2976](https://github.com/GoogleContainerTools/kaniko/pull/2976)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.15.11 to 1.15.14 [#2966](https://github.com/GoogleContainerTools/kaniko/pull/2966)
* chore(deps): bump google.golang.org/api from 0.155.0 to 0.157.0 [#2960](https://github.com/GoogleContainerTools/kaniko/pull/2960)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.26.3 to 1.26.5 [#2963](https://github.com/GoogleContainerTools/kaniko/pull/2963)
* chore(deps): update go-git/go-git, ProtonMail/go-cryto, and cloudflare/circl deps [#2959](https://github.com/GoogleContainerTools/kaniko/pull/2959)
* Update clarification for release.md [#2957](https://github.com/GoogleContainerTools/kaniko/pull/2957)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- Asher
- Bob Du
- dependabot[bot]
- JeromeJu
- Maximilian Hippler
- timbavtbc


# v1.20.0 Release 2024-01-17
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.20.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.20.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.20.0-slim
```

* chore(deps): bump golang.org/x/oauth2 from 0.15.0 to 0.16.0 [#2948](https://github.com/GoogleContainerTools/kaniko/pull/2948)
* chore(deps): bump google-github-actions/auth from 2.0.0 to 2.0.1 [#2947](https://github.com/GoogleContainerTools/kaniko/pull/2947)
* chore(deps): bump golang.org/x/sync from 0.5.0 to 0.6.0 [#2950](https://github.com/GoogleContainerTools/kaniko/pull/2950)
* chore(deps): bump github.com/containerd/containerd from 1.7.11 to 1.7.12 [#2951](https://github.com/GoogleContainerTools/kaniko/pull/2951)
* Prevent extra snapshot with --use-new-run [#2943](https://github.com/GoogleContainerTools/kaniko/pull/2943)
* replace github.com/Azure/azure-storage-blob-go => github.com/Azure/azure-sdk-for-go/sdk/storage/azblob [#2945](https://github.com/GoogleContainerTools/kaniko/pull/2945)
* Fixed wrong example in README.md [#2931](https://github.com/GoogleContainerTools/kaniko/pull/2931)
* chore(deps): bump golang.org/x/sys from 0.15.0 to 0.16.0 [#2936](https://github.com/GoogleContainerTools/kaniko/pull/2936)
* chore(deps): bump google.golang.org/api from 0.154.0 to 0.155.0 [#2937](https://github.com/GoogleContainerTools/kaniko/pull/2937)
* chore(deps): bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 [#2942](https://github.com/GoogleContainerTools/kaniko/pull/2942)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.15.9 to 1.15.11 [#2939](https://github.com/GoogleContainerTools/kaniko/pull/2939)
* chore(deps): bump AdityaGarg8/remove-unwanted-software from 1 to 2 [#2940](https://github.com/GoogleContainerTools/kaniko/pull/2940)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.47.7 to 1.47.8 [#2932](https://github.com/GoogleContainerTools/kaniko/pull/2932)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.26.2 to 1.26.3 [#2933](https://github.com/GoogleContainerTools/kaniko/pull/2933)
* chore(deps): bump github.com/google/go-containerregistry from 0.15.2 to 0.17.0 [#2924](https://github.com/GoogleContainerTools/kaniko/pull/2924)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.15.7 to 1.15.9 [#2926](https://github.com/GoogleContainerTools/kaniko/pull/2926)
* chore(deps): bump google-github-actions/setup-gcloud from 2.0.0 to 2.0.1 [#2927](https://github.com/GoogleContainerTools/kaniko/pull/2927)


Huge thank you for this release towards our contributors: 
- Asher
- Bob Du
- dependabot[bot]
- Maximilian Hippler


# v1.19.2 Release 2023-12-19
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.19.2
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.19.2-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.19.2-slim
```

* chore: update gcr and acr cred helpers [#2910](https://github.com/GoogleContainerTools/kaniko/pull/2910)
* chore(deps): bump sigstore/cosign-installer from 3.2.0 to 3.3.0 [#2911](https://github.com/GoogleContainerTools/kaniko/pull/2911)
* chore(deps): bump google.golang.org/api from 0.152.0 to 0.154.0 [#2912](https://github.com/GoogleContainerTools/kaniko/pull/2912)
* chore(deps): bump cloud.google.com/go/storage from 1.35.1 to 1.36.0 [#2913](https://github.com/GoogleContainerTools/kaniko/pull/2913)
* chore(deps): bump github.com/spf13/cobra from 1.7.0 to 1.8.0 [#2914](https://github.com/GoogleContainerTools/kaniko/pull/2914)
* chore(deps): bump golang.org/x/crypto from 0.16.0 to 0.17.0 [#2915](https://github.com/GoogleContainerTools/kaniko/pull/2915)
* fix: resolve integration test issue issue where container-diff cannot pull OCI images properly from registry [#2918](https://github.com/GoogleContainerTools/kaniko/pull/2918)
* fix: also update github.com/awslabs/amazon-ecr-credential-helper to resolve issues with AWS ECR authentication (resolves #2882) [#2908](https://github.com/GoogleContainerTools/kaniko/pull/2908)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- Patrick Decat


# v1.19.1 Release 2023-12-15
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.19.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.19.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.19.1-slim
```

* Reproducing and Fixing #2892 [#2893](https://github.com/GoogleContainerTools/kaniko/pull/2893)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.15.3 to 1.15.7 [#2897](https://github.com/GoogleContainerTools/kaniko/pull/2897)
* chore(deps): bump google-github-actions/setup-gcloud from 1.1.1 to 2.0.0 [#2902](https://github.com/GoogleContainerTools/kaniko/pull/2902)
* chore(deps): bump actions/setup-go from 4.1.0 to 5.0.0 [#2901](https://github.com/GoogleContainerTools/kaniko/pull/2901)
* chore(deps): bump github.com/containerd/containerd from 1.7.10 to 1.7.11 [#2899](https://github.com/GoogleContainerTools/kaniko/pull/2899)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.10.1 to 5.11.0 [#2898](https://github.com/GoogleContainerTools/kaniko/pull/2898)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.23.5 to 1.24.0 [#2896](https://github.com/GoogleContainerTools/kaniko/pull/2896)
* chore(deps): bump github.com/containerd/containerd from 1.7.9 to 1.7.10 [#2888](https://github.com/GoogleContainerTools/kaniko/pull/2888)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.25.8 to 1.25.11 [#2889](https://github.com/GoogleContainerTools/kaniko/pull/2889)
* chore(deps): bump google-github-actions/auth from 1.2.0 to 2.0.0 [#2886](https://github.com/GoogleContainerTools/kaniko/pull/2886)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.10.0 to 5.10.1 [#2890](https://github.com/GoogleContainerTools/kaniko/pull/2890)
* fix: resolve aws-sdk-go-v2 lib compat issues causing ECR failures [#2885](https://github.com/GoogleContainerTools/kaniko/pull/2885)
* chore(deps): bump github.com/spf13/afero from 1.10.0 to 1.11.0 [#2891](https://github.com/GoogleContainerTools/kaniko/pull/2891)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- Maxime BOSSARD


# v1.19.0 Release 2023-11-29
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.19.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.19.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.19.0-slim
```

* fix: resolve issue with copy_multistage_test.go and broken ioutil import [#2879](https://github.com/GoogleContainerTools/kaniko/pull/2879)
* Fix warmer memory leak. [#2763](https://github.com/GoogleContainerTools/kaniko/pull/2763)
* Skip the /kaniko directory when copying root [#2863](https://github.com/GoogleContainerTools/kaniko/pull/2863)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.25.5 to 1.25.8 [#2875](https://github.com/GoogleContainerTools/kaniko/pull/2875)
* fix: Remove references to deprecated io/ioutil pkg [#2867](https://github.com/GoogleContainerTools/kaniko/pull/2867)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.14.0 to 1.14.3 [#2874](https://github.com/GoogleContainerTools/kaniko/pull/2874)
* Create intermediate directories in COPY with correct uid and gid [#2795](https://github.com/GoogleContainerTools/kaniko/pull/2795)
* chore(deps): bump google-github-actions/auth from 1.1.1 to 1.2.0 [#2868](https://github.com/GoogleContainerTools/kaniko/pull/2868)
* chore(deps): bump golang.org/x/oauth2 from 0.13.0 to 0.14.0 [#2871](https://github.com/GoogleContainerTools/kaniko/pull/2871)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.43.0 to 1.44.0 [#2872](https://github.com/GoogleContainerTools/kaniko/pull/2872)
* chore(deps): bump github.com/containerd/containerd from 1.7.8 to 1.7.9 [#2873](https://github.com/GoogleContainerTools/kaniko/pull/2873)
* impl: add a retry with result function (#2837) [#2853](https://github.com/GoogleContainerTools/kaniko/pull/2853)
* chore(deps): bump docker/build-push-action from 5.0.0 to 5.1.0 [#2857](https://github.com/GoogleContainerTools/kaniko/pull/2857)
* chore(deps): bump golang.org/x/net from 0.17.0 to 0.18.0 [#2859](https://github.com/GoogleContainerTools/kaniko/pull/2859)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.13.1 to 1.14.0 [#2861](https://github.com/GoogleContainerTools/kaniko/pull/2861)
* chore(deps): bump google.golang.org/api from 0.150.0 to 0.151.0 [#2862](https://github.com/GoogleContainerTools/kaniko/pull/2862)
* fix: makefile container-diff on darwin [#2842](https://github.com/GoogleContainerTools/kaniko/pull/2842)
* Print error to stderr instead of stdout before exiting [#2823](https://github.com/GoogleContainerTools/kaniko/pull/2823)
* refactor: rm bool param detectFilesystem in `InitIgnoreList` [#2843](https://github.com/GoogleContainerTools/kaniko/pull/2843)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.22.0 to 1.24.0 [#2851](https://github.com/GoogleContainerTools/kaniko/pull/2851)
* chore(deps): bump google.golang.org/api from 0.149.0 to 0.150.0 [#2845](https://github.com/GoogleContainerTools/kaniko/pull/2845)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.22.1 to 1.22.2 [#2846](https://github.com/GoogleContainerTools/kaniko/pull/2846)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.42.0 to 1.42.1 [#2847](https://github.com/GoogleContainerTools/kaniko/pull/2847)
* chore(deps): bump golang.org/x/sys from 0.13.0 to 0.14.0 [#2848](https://github.com/GoogleContainerTools/kaniko/pull/2848)
* chore(deps): bump sigstore/cosign-installer from 3.1.2 to 3.2.0 [#2849](https://github.com/GoogleContainerTools/kaniko/pull/2849)
* feat: support https URLs for digest-file [#2811](https://github.com/GoogleContainerTools/kaniko/pull/2811)
* impl: add a retry with result function [#2837](https://github.com/GoogleContainerTools/kaniko/pull/2837)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- Adrià Garriga-Alonso
- Anna Levenberg
- Anoop S
- dependabot[bot]
- JeromeJu
- Lio李歐
- Manish Giri
- Maxime BOSSARD
- tal66


# v1.18.0 Release 2023-11-07
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.18.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.18.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.18.0-slim
```

* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.92 to 1.13.1 [#2829](https://github.com/GoogleContainerTools/kaniko/pull/2829)
* chore(deps): bump google.golang.org/api from 0.148.0 to 0.149.0 [#2831](https://github.com/GoogleContainerTools/kaniko/pull/2831)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.40.2 to 1.42.0 [#2828](https://github.com/GoogleContainerTools/kaniko/pull/2828)
* chore(deps): bump golang.org/x/sync from 0.4.0 to 0.5.0 [#2827](https://github.com/GoogleContainerTools/kaniko/pull/2827)
* fix: fix COPY command error due to missing but ignored files [#2812](https://github.com/GoogleContainerTools/kaniko/pull/2812)
* snapshotter: use syncfs system call [#2816](https://github.com/GoogleContainerTools/kaniko/pull/2816)
* Fix missing slash [#2658](https://github.com/GoogleContainerTools/kaniko/pull/2658)
* chore(deps): bump github.com/containerd/containerd from 1.7.7 to 1.7.8 [#2819](https://github.com/GoogleContainerTools/kaniko/pull/2819)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.9.0 to 5.10.0 [#2818](https://github.com/GoogleContainerTools/kaniko/pull/2818)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.91 to 1.11.92 [#2814](https://github.com/GoogleContainerTools/kaniko/pull/2814)
* chore(deps): bump google.golang.org/api from 0.145.0 to 0.148.0 [#2810](https://github.com/GoogleContainerTools/kaniko/pull/2810)


Huge thank you for this release towards our contributors: 
- dependabot[bot]
- Paolo Di Tommaso
- Quan Zhang
- zhouhaibing089


# v1.17.0 Release 2023-10-18
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.17.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.17.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.17.0-slim
```

* docs: fix readme sample typo [#2792](https://github.com/GoogleContainerTools/kaniko/pull/2792)
* fix: remove log line from listpullreqs.go and additional release.sh fixes [#2790](https://github.com/GoogleContainerTools/kaniko/pull/2790)
* chore(deps): bump golang.org/x/sync from 0.3.0 to 0.4.0 [#2798](https://github.com/GoogleContainerTools/kaniko/pull/2798)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.87 to 1.11.91 [#2805](https://github.com/GoogleContainerTools/kaniko/pull/2805)
* chore(deps): bump github.com/containerd/containerd from 1.7.6 to 1.7.7 [#2797](https://github.com/GoogleContainerTools/kaniko/pull/2797)
* chore(deps): bump github.com/google/go-cmp from 0.5.9 to 0.6.0 [#2796](https://github.com/GoogleContainerTools/kaniko/pull/2796)
* chore(deps): bump golang.org/x/net from 0.16.0 to 0.17.0 [#2791](https://github.com/GoogleContainerTools/kaniko/pull/2791)
* fix: resolve issue with integration tests where lack of disk space caused k3s issues [#2804](https://github.com/GoogleContainerTools/kaniko/pull/2804)
* test: add test cases and docString for regex in COPY command [#2773](https://github.com/GoogleContainerTools/kaniko/pull/2773)
* feat: add automated way of cutting releases w/ generation of CHANGELOG.md {{PULL_REQUESTS}} Makefile changes [#2786](https://github.com/GoogleContainerTools/kaniko/pull/2786)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.40.0 to 1.40.1 [#2780](https://github.com/GoogleContainerTools/kaniko/pull/2780)
* docs: Update designdoc.md with correct link to skaffold repository [#2775](https://github.com/GoogleContainerTools/kaniko/pull/2775)
* chore(deps): bump google.golang.org/api from 0.143.0 to 0.145.0 [#2778](https://github.com/GoogleContainerTools/kaniko/pull/2778)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.18.42 to 1.18.44 [#2777](https://github.com/GoogleContainerTools/kaniko/pull/2777)
* chore(deps): bump golang.org/x/oauth2 from 0.12.0 to 0.13.0 [#2781](https://github.com/GoogleContainerTools/kaniko/pull/2781)
* refactor: Remove fallbackToUID bool option from Kaniko code [#2767](https://github.com/GoogleContainerTools/kaniko/pull/2767)
* chore(deps): bump github.com/otiai10/copy from 1.12.0 to 1.14.0 [#2772](https://github.com/GoogleContainerTools/kaniko/pull/2772)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.86 to 1.11.87 [#2770](https://github.com/GoogleContainerTools/kaniko/pull/2770)
* chore(deps): bump google.golang.org/api from 0.142.0 to 0.143.0 [#2769](https://github.com/GoogleContainerTools/kaniko/pull/2769)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.39.0 to 1.40.0 [#2771](https://github.com/GoogleContainerTools/kaniko/pull/2771)
* chore(deps): bump github.com/spf13/afero from 1.9.5 to 1.10.0 [#2758](https://github.com/GoogleContainerTools/kaniko/pull/2758)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.83 to 1.11.86 [#2757](https://github.com/GoogleContainerTools/kaniko/pull/2757)
* chore(deps): bump google.golang.org/api from 0.141.0 to 0.142.0 [#2756](https://github.com/GoogleContainerTools/kaniko/pull/2756)


Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- JeromeJu
- Vishal Khot
- vivekkoya
- zhangzhiqiangcs


# v1.16.0 Release 2023-09-22
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.16.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.16.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.16.0-slim
```

* fix: make it so release.sh script doesn't output duplicate change PRs [#2735](https://github.com/GoogleContainerTools/kaniko/pull/2735)
* chore: update function names to be correct and representative of functionality [#2720](https://github.com/GoogleContainerTools/kaniko/pull/2720)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.8.1 to 5.9.0 [#2749](https://github.com/GoogleContainerTools/kaniko/pull/2749)
* chore(deps): bump google.golang.org/api from 0.140.0 to 0.141.0 [#2748](https://github.com/GoogleContainerTools/kaniko/pull/2748)
* chore(deps): bump github.com/containerd/containerd from 1.7.5 to 1.7.6 [#2750](https://github.com/GoogleContainerTools/kaniko/pull/2750)
* fix: ensure images layers correspond with the image media type [#2719](https://github.com/GoogleContainerTools/kaniko/pull/2719)
* chore(deps): bump github.com/google/slowjam from 1.0.1 to 1.1.0 [#2745](https://github.com/GoogleContainerTools/kaniko/pull/2745)
* chore(deps): bump docker/setup-buildx-action from 2.10.0 to 3.0.0 [#2743](https://github.com/GoogleContainerTools/kaniko/pull/2743)
* chore(deps): bump github.com/go-git/go-billy/v5 from 5.4.1 to 5.5.0 [#2746](https://github.com/GoogleContainerTools/kaniko/pull/2746)
* chore(deps): bump google.golang.org/api from 0.138.0 to 0.140.0 [#2747](https://github.com/GoogleContainerTools/kaniko/pull/2747)
* chore(deps): bump docker/setup-qemu-action from 2.2.0 to 3.0.0 [#2744](https://github.com/GoogleContainerTools/kaniko/pull/2744)
* chore(deps): bump docker/build-push-action from 4.2.1 to 5.0.0 [#2742](https://github.com/GoogleContainerTools/kaniko/pull/2742)
* chore(deps): bump google.golang.org/api from 0.138.0 to 0.139.0 [#2741](https://github.com/GoogleContainerTools/kaniko/pull/2741)
* chore(deps): bump cloud.google.com/go/storage from 1.32.0 to 1.33.0 [#2740](https://github.com/GoogleContainerTools/kaniko/pull/2740)
* chore(deps): bump docker/build-push-action from 4.1.1 to 4.2.1 [#2739](https://github.com/GoogleContainerTools/kaniko/pull/2739)
* chore(deps): bump golang.org/x/oauth2 from 0.11.0 to 0.12.0 [#2732](https://github.com/GoogleContainerTools/kaniko/pull/2732)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.81 to 1.11.83 [#2733](https://github.com/GoogleContainerTools/kaniko/pull/2733)
* chore(deps): bump golang.org/x/net from 0.14.0 to 0.15.0 [#2734](https://github.com/GoogleContainerTools/kaniko/pull/2734)
* chore(deps): bump github.com/containerd/containerd from 1.7.3 to 1.7.5 [#2723](https://github.com/GoogleContainerTools/kaniko/pull/2723)
* chore(deps): bump sigstore/cosign-installer from 3.1.1 to 3.1.2 [#2727](https://github.com/GoogleContainerTools/kaniko/pull/2727)
* chore(deps): bump docker/setup-buildx-action from 2.9.1 to 2.10.0 [#2726](https://github.com/GoogleContainerTools/kaniko/pull/2726)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.18.37 to 1.18.38 [#2724](https://github.com/GoogleContainerTools/kaniko/pull/2724)
* chore(deps): bump golang.org/x/sys from 0.11.0 to 0.12.0 [#2722](https://github.com/GoogleContainerTools/kaniko/pull/2722)
* chore: unnecessary use of fmt.Sprintf [#2717](https://github.com/GoogleContainerTools/kaniko/pull/2717)
* fix function name on comment [#2707](https://github.com/GoogleContainerTools/kaniko/pull/2707)
* Avoid returning the UID when resolving the GIDs. [#2689](https://github.com/GoogleContainerTools/kaniko/pull/2689)

Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- Diego Gonzalez
- geekvest
- guangwu
- Logan Price


# v1.15.0 Release 2023-08-29
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.15.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.15.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.15.0-slim
```

* Ensure New Layers Match Image Media Type [#2700](https://github.com/GoogleContainerTools/kaniko/pull/2700)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.79 to 1.11.81 [#2702](https://github.com/GoogleContainerTools/kaniko/pull/2702)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.38.4 to 1.38.5 [#2706](https://github.com/GoogleContainerTools/kaniko/pull/2706)
* chore(deps): bump google.golang.org/api from 0.136.0 to 0.138.0 [#2704](https://github.com/GoogleContainerTools/kaniko/pull/2704)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.20.3 to 1.21.0 [#2703](https://github.com/GoogleContainerTools/kaniko/pull/2703)
* docs: fix --use-new-run typo [#2698](https://github.com/GoogleContainerTools/kaniko/pull/2698)
* docs: add more information regarding --use-new-run [#2687](https://github.com/GoogleContainerTools/kaniko/pull/2687)
* chore(deps): bump cloud.google.com/go/storage from 1.31.0 to 1.32.0 [#2692](https://github.com/GoogleContainerTools/kaniko/pull/2692)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.77 to 1.11.79 [#2690](https://github.com/GoogleContainerTools/kaniko/pull/2690)
* Fix: Change condition for the behaviour when --no-push=true without setting --destinations [#2676](https://github.com/GoogleContainerTools/kaniko/pull/2676)

Huge thank you for this release towards our contributors: 
- Aaron Prindle
- dependabot[bot]
- JeromeJu
- Logan Price


# v1.14.0 Release 2023-08-15
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.14.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.14.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.14.0-slim
```

* chore(deps): bump actions/setup-go from 4.0.1 to 4.1.0 [#2672](https://github.com/GoogleContainerTools/kaniko/pull/2672)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.18.29 to 1.18.31 [#2651](https://github.com/GoogleContainerTools/kaniko/pull/2651)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.18.31 to 1.18.33 [#2680](https://github.com/GoogleContainerTools/kaniko/pull/2680)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.73 to 1.11.75 [#2650](https://github.com/GoogleContainerTools/kaniko/pull/2650)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.75 to 1.11.77 [#2679](https://github.com/GoogleContainerTools/kaniko/pull/2679)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.37.0 to 1.37.1 [#2648](https://github.com/GoogleContainerTools/kaniko/pull/2648)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.38.0 to 1.38.2 [#2673](https://github.com/GoogleContainerTools/kaniko/pull/2673)
* chore(deps): bump github.com/containerd/containerd from 1.7.2 to 1.7.3 [#2644](https://github.com/GoogleContainerTools/kaniko/pull/2644)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.8.0 to 5.8.1 [#2662](https://github.com/GoogleContainerTools/kaniko/pull/2662)
* chore(deps): bump golang from 1.20 to 1.21 in /deploy [#2682](https://github.com/GoogleContainerTools/kaniko/pull/2682)
* chore(deps): bump golang.org/x/net from 0.12.0 to 0.14.0 [#2663](https://github.com/GoogleContainerTools/kaniko/pull/2663)
* chore(deps): bump golang.org/x/oauth2 from 0.10.0 to 0.11.0 [#2661](https://github.com/GoogleContainerTools/kaniko/pull/2661)
* chore(deps): bump golang.org/x/sys from 0.10.0 to 0.11.0 [#2659](https://github.com/GoogleContainerTools/kaniko/pull/2659)
* chore(deps): bump google.golang.org/api from 0.133.0 to 0.134.0 [#2645](https://github.com/GoogleContainerTools/kaniko/pull/2645)
* chore(deps): bump google.golang.org/api from 0.134.0 to 0.136.0 [#2681](https://github.com/GoogleContainerTools/kaniko/pull/2681)
* docs: add enforcement section to code-of-conduct.md [#2654](https://github.com/GoogleContainerTools/kaniko/pull/2654)
* feat: added skip-push-permission flag [#2657](https://github.com/GoogleContainerTools/kaniko/pull/2657)
* fix: resolve issue where CI env was failing due to dependency change [#2668](https://github.com/GoogleContainerTools/kaniko/pull/2668)
* refactor: Avoid redundant calls to filepath.Clean [#2652](https://github.com/GoogleContainerTools/kaniko/pull/2652)

Huge thank you for this release towards our contributors:
- Aaron Lehmann
- Aaron Prindle
- dependabot[bot]
- Julian


# v1.13.0 Release 2023-07-26
The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.13.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.13.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.13.0-slim
```


* chore(deps): bump cloud.google.com/go/storage from 1.30.1 to 1.31.0 [#2611](https://github.com/GoogleContainerTools/kaniko/pull/2611)
* chore(deps): bump docker/setup-buildx-action from 2.7.0 to 2.8.0 [#2606](https://github.com/GoogleContainerTools/kaniko/pull/2606)
* chore(deps): bump docker/setup-buildx-action from 2.8.0 to 2.9.1 [#2626](https://github.com/GoogleContainerTools/kaniko/pull/2626)
* chore(deps): bump github.com/aws/aws-sdk-go-v2 from 1.18.1 to 1.19.0 [#2623](https://github.com/GoogleContainerTools/kaniko/pull/2623)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.18.28 to 1.18.29 [#2638](https://github.com/GoogleContainerTools/kaniko/pull/2638)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.70 to 1.11.71 [#2610](https://github.com/GoogleContainerTools/kaniko/pull/2610)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.71 to 1.11.72 [#2624](https://github.com/GoogleContainerTools/kaniko/pull/2624)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.72 to 1.11.73 [#2639](https://github.com/GoogleContainerTools/kaniko/pull/2639)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.7.0 to 5.8.0 [#2633](https://github.com/GoogleContainerTools/kaniko/pull/2633)
* chore(deps): bump golang.org/x/oauth2 from 0.9.0 to 0.10.0 [#2617](https://github.com/GoogleContainerTools/kaniko/pull/2617)
* chore(deps): bump golang.org/x/sys from 0.9.0 to 0.10.0 [#2613](https://github.com/GoogleContainerTools/kaniko/pull/2613)
* chore(deps): bump google.golang.org/api from 0.128.0 to 0.129.0 [#2609](https://github.com/GoogleContainerTools/kaniko/pull/2609)
* chore(deps): bump google.golang.org/api from 0.129.0 to 0.131.0 [#2625](https://github.com/GoogleContainerTools/kaniko/pull/2625)
* chore(deps): bump google.golang.org/api from 0.131.0 to 0.132.0 [#2634](https://github.com/GoogleContainerTools/kaniko/pull/2634)
* chore(deps): bump google.golang.org/api from 0.132.0 to 0.133.0 [#2636](https://github.com/GoogleContainerTools/kaniko/pull/2636)
* chore(deps): bump sigstore/cosign-installer from 3.1.0 to 3.1.1 [#2607](https://github.com/GoogleContainerTools/kaniko/pull/2607)
* feat: Allows to disable the fallback to the default registry on image pull [#2637](https://github.com/GoogleContainerTools/kaniko/pull/2637)

Huge thank you for this release towards our contributors: 
- dependabot[bot]
- Fernando Giannetti


# v1.12.1 Release 2023-06-29

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.12.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.12.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.12.1-slim
```

The warmer images are available at:
```
gcr.io/kaniko-project/warmer:v1.12.1
gcr.io/kaniko-project/warmer:latest
```

Fixes:
* fix: resolve issue where warmer CLI always validated optional arg -> breakage for majority of users [#2603](https://github.com/GoogleContainerTools/kaniko/pull/2603)


# v1.12.0 Release 2023-06-28

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.12.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.12.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.12.0-slim
```

* chore: add debug line to RedoHasher [#2591](https://github.com/GoogleContainerTools/kaniko/pull/2591)
* chore(deps): bump docker/build-push-action from 4.0.0 to 4.1.0 [#2557](https://github.com/GoogleContainerTools/kaniko/pull/2557)
* chore(deps): bump docker/build-push-action from 4.1.0 to 4.1.1 [#2580](https://github.com/GoogleContainerTools/kaniko/pull/2580)
* chore(deps): bump docker/setup-buildx-action from 2.5.0 to 2.6.0 [#2555](https://github.com/GoogleContainerTools/kaniko/pull/2555)
* chore(deps): bump docker/setup-buildx-action from 2.6.0 to 2.7.0 [#2579](https://github.com/GoogleContainerTools/kaniko/pull/2579)
* chore(deps): bump docker/setup-qemu-action from 2.1.0 to 2.2.0 [#2556](https://github.com/GoogleContainerTools/kaniko/pull/2556)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/config from 1.18.25 to 1.18.27 [#2581](https://github.com/GoogleContainerTools/kaniko/pull/2581)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/feature/s3/manager from 1.11.67 to 1.11.70 [#2597](https://github.com/GoogleContainerTools/kaniko/pull/2597)
* chore(deps): bump github.com/aws/aws-sdk-go-v2/service/s3 from 1.33.1 to 1.35.0 [#2582](https://github.com/GoogleContainerTools/kaniko/pull/2582)
* chore(deps): bump github.com/otiai10/copy from 1.11.0 to 1.12.0 [#2598](https://github.com/GoogleContainerTools/kaniko/pull/2598)
* chore(deps): bump golang.org/x/oauth2 from 0.8.0 to 0.9.0 [#2578](https://github.com/GoogleContainerTools/kaniko/pull/2578)
* chore(deps): bump golang.org/x/sync from 0.2.0 to 0.3.0 [#2573](https://github.com/GoogleContainerTools/kaniko/pull/2573)
* chore(deps): bump golang.org/x/sys from 0.8.0 to 0.9.0 [#2564](https://github.com/GoogleContainerTools/kaniko/pull/2564)
* chore(deps): bump google.golang.org/api from 0.125.0 to 0
* chore(deps): bump google.golang.org/api from 0.126.0 to 0.127.0 [#2565](https://github.com/GoogleContainerTools/kaniko/pull/2565)
* chore(deps): bump google.golang.org/api from 0.127.0 to 0.128.0 [#2596](https://github.com/GoogleContainerTools/kaniko/pull/2596)
* chore(deps): bump sigstore/cosign-installer from 3.0.5 to 3.1.0 [#2595](https://github.com/GoogleContainerTools/kaniko/pull/2595)
* Don't write whiteout files to directories that were replaced with files or links [#2590](https://github.com/GoogleContainerTools/kaniko/pull/2590)
* feat: cache dockerfile images through warmer [#2499](https://github.com/GoogleContainerTools/kaniko/pull/2499)
* Fix fs_util tests failing on systems with /tmp mountpoint [#2583](https://github.com/GoogleContainerTools/kaniko/pull/2583)
* Fix multistage caching with COPY --from [#2559](https://github.com/GoogleContainerTools/kaniko/pull/2559)
* fix: hack/boilerplate.sh: fix error handling and use python3 [#2587](https://github.com/GoogleContainerTools/kaniko/pull/2587)
* fix: hack/install_golint.sh: allow installation on linux/arm64 [#2585](https://github.com/GoogleContainerTools/kaniko/pull/2585)
* fix: install tools using go.mod for versioning [#2562](https://github.com/GoogleContainerTools/kaniko/pull/2562)
* fix: Refactors IsSrcRemoteFileURL to only validate the URL is valid [#2563](https://github.com/GoogleContainerTools/kaniko/pull/2563)
* fix: update cache-ttl help text to be correct regarding unit of duration [#2568](https://github.com/GoogleContainerTools/kaniko/pull/2568)
* fix: valdiateFlags typo fixed [#2554](https://github.com/GoogleContainerTools/kaniko/pull/2554)

Huge thank you for this release towards our contributors: 
- Aaron Prindle
- alexezio
- Andreas Fleig
- Angus Williams
- dependabot[bot]
- Kraev Sergei
- Liam Newman
- Zigelboim Misha


# v1.11.0 Release 2023-06-08

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.11.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.11.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.11.0-slim
```

* chore: run go mod tidy [#2532](https://github.com/GoogleContainerTools/kaniko/pull/2532)
* chore(deps): bump actions/setup-go from 3.2.0 to 4.0.1 [#2517](https://github.com/GoogleContainerTools/kaniko/pull/2517)
* chore(deps): bump cloud.google.com/go/storage from 1.29.0 to 1.30.1 [#2439](https://github.com/GoogleContainerTools/kaniko/pull/2439)
* chore(deps): bump docker/setup-buildx-action from 2.0.0 to 2.5.0 [#2519](https://github.com/GoogleContainerTools/kaniko/pull/2519)
* chore(deps): bump github.com/containerd/containerd from 1.7.0 to 1.7.1 [#2534](https://github.com/GoogleContainerTools/kaniko/pull/2534)
* chore(deps): bump github.com/containerd/containerd from 1.7.1 to 1.7.2 [#2542](https://github.com/GoogleContainerTools/kaniko/pull/2542)
* chore(deps): bump github.com/go-git/go-git/v5 from 5.4.2 to 5.7.0 [#2528](https://github.com/GoogleContainerTools/kaniko/pull/2528)
* chore(deps): bump github.com/google/go-containerregistry from 0.15.1 to 0.15.2 [#2546](https://github.com/GoogleContainerTools/kaniko/pull/2546)
* chore(deps): bump github.com/moby/buildkit from 0.11.4 to 0.11.6 [#2520](https://github.com/GoogleContainerTools/kaniko/pull/2520)
* chore(deps): bump github.com/sirupsen/logrus from 1.9.2 to 1.9.3 [#2545](https://github.com/GoogleContainerTools/kaniko/pull/2545)
* chore(deps): bump google.golang.org/api from 0.121.0 to 0.124.0 [#2535](https://github.com/GoogleContainerTools/kaniko/pull/2535)
* chore(deps): bump google.golang.org/api from 0.124.0 to 0.125.0 [#2544](https://github.com/GoogleContainerTools/kaniko/pull/2544)
* chore(deps): bump sigstore/cosign-installer from 3.0.3 to 3.0.5 [#2518](https://github.com/GoogleContainerTools/kaniko/pull/2518)
* chore(deps): update docker-credential-* binaries in kaniko images [#2531](https://github.com/GoogleContainerTools/kaniko/pull/2531)
* chore(deps): Update google-github-actions/setup-gcloud to v1.1.1 [#2548](https://github.com/GoogleContainerTools/kaniko/pull/2548)
* chore(deps): use aws-sdk-go-v2 [#2550](https://github.com/GoogleContainerTools/kaniko/pull/2550)
* docs: Add guide on creating multi-arch manifests [#2306](https://github.com/GoogleContainerTools/kaniko/pull/2306)
* docs: update changelog to correct old release tags [#2536](https://github.com/GoogleContainerTools/kaniko/pull/2536)
* fix: Deduplicate paths while saving files for later use [#2504](https://github.com/GoogleContainerTools/kaniko/pull/2504)
* fix: Download docker-credential-gcr from release artifacts [#2540](https://github.com/GoogleContainerTools/kaniko/pull/2540)
* refactor: Use a multistage image to remove all redundancies on Dockerfiles [#2547](https://github.com/GoogleContainerTools/kaniko/pull/2547)
* test: only build for linux/amd64 on PRs [#2460](https://github.com/GoogleContainerTools/kaniko/pull/2460)

Huge thank you for this release towards our contributors: 
- Aaron Prindle
- Bob Du
- dependabot[bot]
- Fedor V
- Ferran Vidal
- Jason Hall
- Jasper Ben Orschulko


# v1.10.0 Release 2023-05-24

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.10.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.10.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.10.0-slim
```

* chore(deps): bump github.com/sirupsen/logrus from 1.9.0 to 1.9.2 [#2522](https://github.com/GoogleContainerTools/kaniko/pull/2522)
* chore(deps): bump github.com/otiai10/copy from 1.7.0 to 1.11.0 [#2523](https://github.com/GoogleContainerTools/kaniko/pull/2523)
* Add mTLS (client cert) registry authentication [#2180](https://github.com/GoogleContainerTools/kaniko/pull/2180)
* chore: Revert "chore(deps): bump google-github-actions/setup-gcloud from 0.5.1 to 1.1.1 (#2502)" [#2524](https://github.com/GoogleContainerTools/kaniko/pull/2524)
* Light editing to scripts in hack/gofmt [#2236](https://github.com/GoogleContainerTools/kaniko/pull/2236)
* chore(deps): bump golang from 1.19 to 1.20 in /deploy [#2388](https://github.com/GoogleContainerTools/kaniko/pull/2388)
* chore(deps): bump imjasonh/setup-crane from 0.1 to 0.3 [#2401](https://github.com/GoogleContainerTools/kaniko/pull/2401)
* chore(deps): bump golang.org/x/sync from 0.1.0 to 0.2.0 [#2497](https://github.com/GoogleContainerTools/kaniko/pull/2497)
* fix: Correct deprecated flags in `README.md` [#2335](https://github.com/GoogleContainerTools/kaniko/pull/2335)
* chore(deps): bump docker/setup-qemu-action from 1.2.0 to 2.1.0 [#2287](https://github.com/GoogleContainerTools/kaniko/pull/2287)
* Delete scorecards-analysis.yml [#2510](https://github.com/GoogleContainerTools/kaniko/pull/2510)
* chore(deps): bump docker/build-push-action from 3.2.0 to 4.0.0 [#2505](https://github.com/GoogleContainerTools/kaniko/pull/2505)
* chore(deps): bump github.com/docker/distribution from 2.8.1+incompatible to 2.8.2+incompatible [#2503](https://github.com/GoogleContainerTools/kaniko/pull/2503)
* chore(deps): bump ossf/scorecard-action from 1.1.1 to 2.1.3 [#2506](https://github.com/GoogleContainerTools/kaniko/pull/2506)
* chore(deps): bump golang.org/x/sys from 0.7.0 to 0.8.0 [#2507](https://github.com/GoogleContainerTools/kaniko/pull/2507)
* chore(deps): bump github.com/google/go-containerregistry from 0.14.0 to 0.15.1 [#2508](https://github.com/GoogleContainerTools/kaniko/pull/2508)
* chore(deps): bump github.com/google/slowjam from 1.0.0 to 1.0.1 [#2498](https://github.com/GoogleContainerTools/kaniko/pull/2498)
* chore(deps): bump google-github-actions/setup-gcloud from 0.5.1 to 1.1.1 [#2502](https://github.com/GoogleContainerTools/kaniko/pull/2502)
* chore: add .vscode/ dir to .gitignore [#2501](https://github.com/GoogleContainerTools/kaniko/pull/2501)
* chore(deps): bump sigstore/cosign-installer from 3.0.1 to 3.0.3 [#2495](https://github.com/GoogleContainerTools/kaniko/pull/2495)
* chore(deps): bump google.golang.org/api from 0.120.0 to 0.121.0 [#2496](https://github.com/GoogleContainerTools/kaniko/pull/2496)
* chore(deps): bump github.com/spf13/afero from 1.9.2 to 1.9.5 [#2448](https://github.com/GoogleContainerTools/kaniko/pull/2448)
* chore(deps): bump google.golang.org/api from 0.110.0 to 0.120.0 [#2484](https://github.com/GoogleContainerTools/kaniko/pull/2484)
* chore(deps): bump github/codeql-action from 2.1.8 to 2.3.2 [#2487](https://github.com/GoogleContainerTools/kaniko/pull/2487)
* chore(deps): bump github.com/docker/docker from 23.0.1+incompatible to 23.0.5+incompatible [#2489](https://github.com/GoogleContainerTools/kaniko/pull/2489)
* chore(deps): bump github.com/aws/aws-sdk-go from 1.44.24 to 1.44.253 [#2490](https://github.com/GoogleContainerTools/kaniko/pull/2490)
* fix: use debian buster to fix tests using no longer supported stretch which had broken apt-get urls [#2485](https://github.com/GoogleContainerTools/kaniko/pull/2485)
* chore(deps): bump google.golang.org/protobuf from 1.29.0 to 1.29.1 [#2442](https://github.com/GoogleContainerTools/kaniko/pull/2442)
* Use correct media type for zstd layers [#2459](https://github.com/GoogleContainerTools/kaniko/pull/2459)
* Add support for zstd compression [#2313](https://github.com/GoogleContainerTools/kaniko/pull/2313)
* chore(deps): bump github.com/opencontainers/runc from 1.1.4 to 1.1.5 [#2453](https://github.com/GoogleContainerTools/kaniko/pull/2453)

Huge thank you for this release towards our contributors: 
- Aaron Prindle
- Aaruni Aggarwal
- Abirdcfly
- Adrian Newby
- almg80
- Anbraten
- Andreas Fleig
- Andrei Kvapil
- ankitm123
- Aris Buzachis
- Benjamin Krenn
- Bernardo Marques
- Bryan A. S
- chenggui53
- Chuang Wang
- claudex
- Dávid Szakállas
- Dawei Ma
- dependabot[bot]
- Diego Gonzalez
- dmr
- ejose19
- Eng Zer Jun
- ePirat
- Eric
- Florian Apolloner
- François JACQUES
- Gabriel Nützi
- Gilbert Gilb's
- Guillaume Calmettes
- Herman
- Hingbong Lo
- Igor Scheller
- Ishant Mrinal Haloi
- Jack
- Jake Sanders
- Janosch Maier
- Jason D'Amour
- Jason Hall
- Jasper Ben Orschulko
- Jerry Jones
- jeunii
- Joe Kimmel
- Joël Pepper
- Jonas Gröger
- Jose Donizetti
- Junwon Kwon
- Kamal Nasser
- Konstantin Demin
- Kun Lu
- Lars Seipel
- Lavrenti Frobeen
- Liwen Guo
- Lukas
- Mark Moretto
- Matt Moore
- Max Walther
- Mikhail Vasin
- Natalie Arellano
- Naveen
- nihilo
- Oliver Gregorius
- Pat Litke
- Patrick Barker
- priyawadhwa
- Ramy
- Rhianna
- Sebastiaan Tammer
- Shude Li
- Sigurd Spieckermann
- Silvano Cirujano Cuesta
- Tejal Desai
- Tony De La Nuez
- Travis DePrato
- Viacheslav Artamonov
- Víctor
- Wolfgang Walther
- wwade
- Yahav Itzhak
- ygelfand
- Yonatan Koren
- zhouhaibing089

# v1.9.2 Release 2023-03-27

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.9.2
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.9.2-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.9.2-slim
```

* fix(executor): make pax tar builds reproducible again [#2384](https://github.com/GoogleContainerTools/kaniko/pull/2384)
* Upgrade docker [#2440](https://github.com/GoogleContainerTools/kaniko/pull/2440)
* Update ACR credential helper to enable Azure Workload Identity [#2431](https://github.com/GoogleContainerTools/kaniko/pull/2431)
* bump cosign version used to sign images [#2437](https://github.com/GoogleContainerTools/kaniko/pull/2437)
* Fix Integration tests [#2425](https://github.com/GoogleContainerTools/kaniko/pull/2425)
* chore(deps): bump golang from 1.17 to 1.19 in /deploy [#2328](https://github.com/GoogleContainerTools/kaniko/pull/2328)
* chore: fix typo [#2316](https://github.com/GoogleContainerTools/kaniko/pull/2316)
* ci: don't cache certs stage [#2296](https://github.com/GoogleContainerTools/kaniko/pull/2296)
* fix(executor): make pax tar builds reproducible again [#2384](https://github.com/GoogleContainerTools/kaniko/pull/2384)
* Upgrade docker [#2440](https://github.com/GoogleContainerTools/kaniko/pull/2440)
* Update ACR credential helper to enable Azure Workload Identity [#2431](https://github.com/GoogleContainerTools/kaniko/pull/2431)
* bump cosign version used to sign images [#2437](https://github.com/GoogleContainerTools/kaniko/pull/2437)
* Fix Integration tests [#2425](https://github.com/GoogleContainerTools/kaniko/pull/2425)
* chore(deps): bump golang from 1.17 to 1.19 in /deploy [#2328](https://github.com/GoogleContainerTools/kaniko/pull/2328)
* chore: fix typo [#2316](https://github.com/GoogleContainerTools/kaniko/pull/2316)
* ci: don't cache certs stage [#2296](https://github.com/GoogleContainerTools/kaniko/pull/2296)
* chore: fix typo [#2289](https://github.com/GoogleContainerTools/kaniko/pull/2289)
* fix(WORKDIR): use the config.User for the new dir permissions [#2269](https://github.com/GoogleContainerTools/kaniko/pull/2269)
* Provide `--cache-repo` as OCI image layout path [#2250](https://github.com/GoogleContainerTools/kaniko/pull/2250)
Huge thank you for this release towards our contributors: 
- Aaruni Aggarwal
- Abirdcfly
- Adrian Newby
- almg80
- Anbraten
- Andreas Fleig
- Andrei Kvapil
- ankitm123
- Aris Buzachis
- Benjamin Krenn
- Bernardo Marques
- Bryan A. S
- chenggui53
- Chuang Wang
- claudex
- Dávid Szakállas
- Dawei Ma
- dependabot[bot]
- Diego Gonzalez
- dmr
- ejose19
- Eng Zer Jun
- ePirat
- Florian Apolloner
- François JACQUES
- Gabriel Nützi
- Gilbert Gilb's
- Guillaume Calmettes
- Herman
- Hingbong Lo
- Igor Scheller
- Ishant Mrinal Haloi
- Jack
- Jake Sanders
- Janosch Maier
- Jason D'Amour
- Jason Hall
- Jasper Ben Orschulko
- Jerry Jones
- jeunii
- Joe Kimmel
- Joël Pepper
- Jonas Gröger
- Jose Donizetti
- Junwon Kwon
- Kamal Nasser
- Konstantin Demin
- Kun Lu
- Lars Seipel
- Liwen Guo
- Lukas
- Matt Moore
- Max Walther
- Mikhail Vasin
- Natalie Arellano
- Naveen
- nihilo
- Oliver Gregorius
- Pat Litke
- Patrick Barker
- priyawadhwa
- Ramy
- Rhianna
- Sebastiaan Tammer
- Shude Li
- Sigurd Spieckermann
- Silvano Cirujano Cuesta
- Tejal Desai
- Tony De La Nuez
- Travis DePrato
- Viacheslav Artamonov
- Víctor
- Wolfgang Walther
- wwade
- Yahav Itzhak
- ygelfand
- Yonatan Koren
- zhouhaibing089

# v1.9.1 Release 2022-09-26

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.9.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.9.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.9.1-slim
```


* (fix):Pass full URI path to `bucket.GetNameAndFilepathFromURI` [#2221](https://github.com/GoogleContainerTools/kaniko/pull/2221)
* Add the ability to skip unpacking the initial file system [#2234](https://github.com/GoogleContainerTools/kaniko/pull/2234)
* chore: remove duplicate word in comments [#2232](https://github.com/GoogleContainerTools/kaniko/pull/2232)
* docs(CHANGELOG.md): fix link to issue #2040 [#2228](https://github.com/GoogleContainerTools/kaniko/pull/2228)
* feat: disable cache-copy-layers in multistage builds; closes 2065 [#2227](https://github.com/GoogleContainerTools/kaniko/pull/2227)
* bump cosign version so it can sign [#2224](https://github.com/GoogleContainerTools/kaniko/pull/2224)
* fix(README.md): remove duplicate caching section [#2223](https://github.com/GoogleContainerTools/kaniko/pull/2223)
* refactor: Make CLI argument names consistent [#2084](https://github.com/GoogleContainerTools/kaniko/pull/2084)
* fix(KanikoDir): update DOCKER_CONFIG env when use custom kanikoDir [#2202](https://github.com/GoogleContainerTools/kaniko/pull/2202)
* (fix):Pass full URI path to `bucket.GetNameAndFilepathFromURI` [#2221](https://github.com/GoogleContainerTools/kaniko/pull/2221)
* Add the ability to skip unpacking the initial file system [#2234](https://github.com/GoogleContainerTools/kaniko/pull/2234)
* chore: remove duplicate word in comments [#2232](https://github.com/GoogleContainerTools/kaniko/pull/2232)
* docs(CHANGELOG.md): fix link to issue #2040 [#2228](https://github.com/GoogleContainerTools/kaniko/pull/2228)
* feat: disable cache-copy-layers in multistage builds; closes 2065 [#2227](https://github.com/GoogleContainerTools/kaniko/pull/2227)
* bump cosign version so it can sign [#2224](https://github.com/GoogleContainerTools/kaniko/pull/2224)
* fix(README.md): remove duplicate caching section [#2223](https://github.com/GoogleContainerTools/kaniko/pull/2223)
* refactor: Make CLI argument names consistent [#2084](https://github.com/GoogleContainerTools/kaniko/pull/2084)
* fix(KanikoDir): update DOCKER_CONFIG env when use custom kanikoDir [#2202](https://github.com/GoogleContainerTools/kaniko/pull/2202)
Huge thank you for this release towards our contributors: 
- Aaruni Aggarwal
- Abirdcfly
- Adrian Newby
- almg80
- Anbraten
- Andreas Fleig
- Andrei Kvapil
- ankitm123
- Benjamin Krenn
- Bernardo Marques
- Bryan A. S
- chenggui53
- Chuang Wang
- claudex
- Dávid Szakállas
- Dawei Ma
- dependabot[bot]
- Diego Gonzalez
- dmr
- ejose19
- Eng Zer Jun
- ePirat
- Florian Apolloner
- François JACQUES
- Gabriel Nützi
- Gilbert Gilb's
- Guillaume Calmettes
- Herman
- Hingbong Lo
- Igor Scheller
- Ishant Mrinal Haloi
- Jack
- Jake Sanders
- Janosch Maier
- Jason D'Amour
- Jason Hall
- Jasper Ben Orschulko
- jeunii
- Jonas Gröger
- Jose Donizetti
- Kamal Nasser
- Konstantin Demin
- Kun Lu
- Lars Seipel
- Liwen Guo
- Lukas
- Matt Moore
- Max Walther
- Mikhail Vasin
- Natalie Arellano
- Naveen
- nihilo
- Oliver Gregorius
- Pat Litke
- Patrick Barker
- priyawadhwa
- Ramy
- Rhianna
- Sebastiaan Tammer
- Sigurd Spieckermann
- Silvano Cirujano Cuesta
- Tejal Desai
- Tony De La Nuez
- Travis DePrato
- Víctor
- Wolfgang Walther
- wwade
- Yahav Itzhak
- ygelfand
- Yonatan Koren
- zhouhaibing089

# v1.9.0 Release 2022-08-09

## Highlights
- Installed binaries are missing from image [#2049](https://github.com/GoogleContainerTools/kaniko/issues/2049)
- proc: detect kubernetes runtime by mounts [#2054](https://github.com/GoogleContainerTools/kaniko/pull/2054)
- Fixes #2046: make target stage lookup case insensitive [#2047](https://github.com/GoogleContainerTools/kaniko/pull/2047)
- fix: Refactor LayersMap to correct old strange code behavior [#2066](https://github.com/GoogleContainerTools/kaniko/pull/2066)
- Fix missing setuid flags on COPY --from=build operation [#2089](https://github.com/GoogleContainerTools/kaniko/pull/2089)
- Fixes #2046: make target stage lookup case insensitive [#2047](https://github.com/GoogleContainerTools/kaniko/pull/2047)
- Add GitLab CI credentials helper [#2040](https://github.com/GoogleContainerTools/kaniko/pull/2040)
- and a number of dependency bumps



The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.9.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.9.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.9.0-slim
```

* add cache option for run command [#2032](https://github.com/GoogleContainerTools/kaniko/pull/2032)
* fix: kaniko dir env unused [#2067](https://github.com/GoogleContainerTools/kaniko/pull/2067)
* fix: getUIDandGID is able to resolve non-existing users and groups [#2106](https://github.com/GoogleContainerTools/kaniko/pull/2106)
* fix(Dockerfile): use temporary busybox mount to create /kaniko directory [#2155](https://github.com/GoogleContainerTools/kaniko/pull/2155)
* Fix the /kaniko directory permissions in container [#2009](https://github.com/GoogleContainerTools/kaniko/pull/2009)
* ci(setup-minikube): use cri-dockerd [#2149](https://github.com/GoogleContainerTools/kaniko/pull/2149)
* CA certificates tasks in kaniko images [#2142](https://github.com/GoogleContainerTools/kaniko/pull/2142)
* refactor: simpler local integration tests [#2110](https://github.com/GoogleContainerTools/kaniko/pull/2110)
* fix: use refrence should after err handles [#2128](https://github.com/GoogleContainerTools/kaniko/pull/2128)
* fix: Add test for issue #2049 [#2114](https://github.com/GoogleContainerTools/kaniko/pull/2114)
* Bump ossf/scorecard-action from 1.0.4 to 1.1.1 [#2116](https://github.com/GoogleContainerTools/kaniko/pull/2116)
* Bump github.com/aws/aws-sdk-go from 1.43.36 to 1.44.24 [#2111](https://github.com/GoogleContainerTools/kaniko/pull/2111)
* Bump actions/setup-go from 3.0.0 to 3.2.0 [#2112](https://github.com/GoogleContainerTools/kaniko/pull/2112)
* Write parent directories to tar before whiteout files [#2113](https://github.com/GoogleContainerTools/kaniko/pull/2113)
* fix(ci): Docker build for issue 1837 [#2095](https://github.com/GoogleContainerTools/kaniko/pull/2095)
* Update Azure credHelpers docs [#2109](https://github.com/GoogleContainerTools/kaniko/pull/2109)
* Fix missing setuid flags on COPY --from=build operation [#2089](https://github.com/GoogleContainerTools/kaniko/pull/2089)
* fix: `COPY --chown` regression tests [#2097](https://github.com/GoogleContainerTools/kaniko/pull/2097)
* fix: Regression test for #2066 [#2096](https://github.com/GoogleContainerTools/kaniko/pull/2096)
* fix: Refactor `LayersMap` to correct old strange code behavior [#2066](https://github.com/GoogleContainerTools/kaniko/pull/2066)
* fix: Main [#2094](https://github.com/GoogleContainerTools/kaniko/pull/2094)
* feat: add flag to disable pushing cache [#2038](https://github.com/GoogleContainerTools/kaniko/pull/2038)
* hasher: hash security.capability attributes [#1994](https://github.com/GoogleContainerTools/kaniko/pull/1994)
* Documentation: Clarify README.md blurb on `--cache-copy-layers` [#2064](https://github.com/GoogleContainerTools/kaniko/pull/2064)
* Fix release tagging workflow [#2034](https://github.com/GoogleContainerTools/kaniko/pull/2034)
* Bump docker/setup-buildx-action from 1.6.0 to 2 [#2081](https://github.com/GoogleContainerTools/kaniko/pull/2081)
* Bump go-containerregistry dependency [#2076](https://github.com/GoogleContainerTools/kaniko/pull/2076)
* Fix: Flatten layer function needs to return existing files in the layer correctly [#2057](https://github.com/GoogleContainerTools/kaniko/pull/2057)
* fix: Remove hardcoded whiteout prefix [#2056](https://github.com/GoogleContainerTools/kaniko/pull/2056)
* proc: detect kubernetes runtime by mounts [#2054](https://github.com/GoogleContainerTools/kaniko/pull/2054)
* Fixes #2046: make target stage lookup case insensitive [#2047](https://github.com/GoogleContainerTools/kaniko/pull/2047)
* Add GitLab CI credentials helper [#2040](https://github.com/GoogleContainerTools/kaniko/pull/2040)
* Bump sigstore/cosign-installer from b4f55743d10d066fee1de1cf0fa26069700c0195 to 2.2.0 [#2044](https://github.com/GoogleContainerTools/kaniko/pull/2044)
* Bump github/codeql-action from 2.1.6 to 2.1.8 [#2043](https://github.com/GoogleContainerTools/kaniko/pull/2043)
* Bump github.com/aws/aws-sdk-go from 1.43.31 to 1.43.36 [#2042](https://github.com/GoogleContainerTools/kaniko/pull/2042)
* Bump cloud.google.com/go/storage from 1.21.0 to 1.22.0 [#2041](https://github.com/GoogleContainerTools/kaniko/pull/2041)
* add cache option for run command [#2032](https://github.com/GoogleContainerTools/kaniko/pull/2032)
* fix: kaniko dir env unused [#2067](https://github.com/GoogleContainerTools/kaniko/pull/2067)
* fix: getUIDandGID is able to resolve non-existing users and groups [#2106](https://github.com/GoogleContainerTools/kaniko/pull/2106)
* fix(Dockerfile): use temporary busybox mount to create /kaniko directory [#2155](https://github.com/GoogleContainerTools/kaniko/pull/2155)
* Fix the /kaniko directory permissions in container [#2009](https://github.com/GoogleContainerTools/kaniko/pull/2009)
* ci(setup-minikube): use cri-dockerd [#2149](https://github.com/GoogleContainerTools/kaniko/pull/2149)
* CA certificates tasks in kaniko images [#2142](https://github.com/GoogleContainerTools/kaniko/pull/2142)
* refactor: simpler local integration tests [#2110](https://github.com/GoogleContainerTools/kaniko/pull/2110)
* fix: use refrence should after err handles [#2128](https://github.com/GoogleContainerTools/kaniko/pull/2128)
* fix: Add test for issue #2049 [#2114](https://github.com/GoogleContainerTools/kaniko/pull/2114)
* Bump ossf/scorecard-action from 1.0.4 to 1.1.1 [#2116](https://github.com/GoogleContainerTools/kaniko/pull/2116)
* Bump github.com/aws/aws-sdk-go from 1.43.36 to 1.44.24 [#2111](https://github.com/GoogleContainerTools/kaniko/pull/2111)
* Bump actions/setup-go from 3.0.0 to 3.2.0 [#2112](https://github.com/GoogleContainerTools/kaniko/pull/2112)
* Write parent directories to tar before whiteout files [#2113](https://github.com/GoogleContainerTools/kaniko/pull/2113)
* fix(ci): Docker build for issue 1837 [#2095](https://github.com/GoogleContainerTools/kaniko/pull/2095)
* Update Azure credHelpers docs [#2109](https://github.com/GoogleContainerTools/kaniko/pull/2109)
* Fix missing setuid flags on COPY --from=build operation [#2089](https://github.com/GoogleContainerTools/kaniko/pull/2089)
* fix: `COPY --chown` regression tests [#2097](https://github.com/GoogleContainerTools/kaniko/pull/2097)
* fix: Regression test for #2066 [#2096](https://github.com/GoogleContainerTools/kaniko/pull/2096)
* fix: Refactor `LayersMap` to correct old strange code behavior [#2066](https://github.com/GoogleContainerTools/kaniko/pull/2066)
* fix: Main [#2094](https://github.com/GoogleContainerTools/kaniko/pull/2094)
* feat: add flag to disable pushing cache [#2038](https://github.com/GoogleContainerTools/kaniko/pull/2038)
* hasher: hash security.capability attributes [#1994](https://github.com/GoogleContainerTools/kaniko/pull/1994)
* Documentation: Clarify README.md blurb on `--cache-copy-layers` [#2064](https://github.com/GoogleContainerTools/kaniko/pull/2064)
* Fix release tagging workflow [#2034](https://github.com/GoogleContainerTools/kaniko/pull/2034)
* Bump docker/setup-buildx-action from 1.6.0 to 2 [#2081](https://github.com/GoogleContainerTools/kaniko/pull/2081)
* Bump go-containerregistry dependency [#2076](https://github.com/GoogleContainerTools/kaniko/pull/2076)
* Fix: Flatten layer function needs to return existing files in the layer correctly [#2057](https://github.com/GoogleContainerTools/kaniko/pull/2057)
* fix: Remove hardcoded whiteout prefix [#2056](https://github.com/GoogleContainerTools/kaniko/pull/2056)
* proc: detect kubernetes runtime by mounts [#2054](https://github.com/GoogleContainerTools/kaniko/pull/2054)
* Fixes #2046: make target stage lookup case insensitive [#2047](https://github.com/GoogleContainerTools/kaniko/pull/2047)
* Add GitLab CI credentials helper [#2040](https://github.com/GoogleContainerTools/kaniko/pull/2040)
* Bump sigstore/cosign-installer from b4f55743d10d066fee1de1cf0fa26069700c0195 to 2.2.0 [#2044](https://github.com/GoogleContainerTools/kaniko/pull/2044)
* Bump github/codeql-action from 2.1.6 to 2.1.8 [#2043](https://github.com/GoogleContainerTools/kaniko/pull/2043)
* Bump github.com/aws/aws-sdk-go from 1.43.31 to 1.43.36 [#2042](https://github.com/GoogleContainerTools/kaniko/pull/2042)
* Bump cloud.google.com/go/storage from 1.21.0 to 1.22.0 [#2041](https://github.com/GoogleContainerTools/kaniko/pull/2041)
Huge thank you for this release towards our contributors: 
- Aaruni Aggarwal
- Adrian Newby
- Anbraten
- Andreas Fleig
- Andrei Kvapil
- ankitm123
- Benjamin Krenn
- Bernardo Marques
- Chuang Wang
- claudex
- Dávid Szakállas
- Dawei Ma
- dependabot[bot]
- Diego Gonzalez
- ejose19
- Eng Zer Jun
- ePirat
- Florian Apolloner
- François JACQUES
- Gabriel Nützi
- Gilbert Gilb's
- Guillaume Calmettes
- Herman
- Hingbong Lo
- Igor Scheller
- Ishant Mrinal Haloi
- Jack
- Jake Sanders
- Janosch Maier
- Jason D'Amour
- Jason Hall
- Jasper Ben Orschulko
- jeunii
- Jose Donizetti
- Kamal Nasser
- Konstantin Demin
- Kun Lu
- Lars Seipel
- Liwen Guo
- Lukas
- Matt Moore
- Max Walther
- Mikhail Vasin
- Naveen
- nihilo
- Oliver Gregorius
- Pat Litke
- Patrick Barker
- priyawadhwa
- Ramy
- Rhianna
- Sebastiaan Tammer
- Sigurd Spieckermann
- Silvano Cirujano Cuesta
- Tejal Desai
- Tony De La Nuez
- Travis DePrato
- Víctor
- Wolfgang Walther
- wwade
- Yahav Itzhak
- ygelfand
- Yonatan Koren
- zhouhaibing089

# v1.8.1 Release 2022-04-01
This is Apr's 2022 release.

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.8.1
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.8.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.8.1-slim
```

* Use canonical platform values. Fix 1995. [#2025](https://github.com/GoogleContainerTools/kaniko/pull/2025)
* feat: kaniko dir config option [#1997](https://github.com/GoogleContainerTools/kaniko/pull/1997)
* Bump github.com/aws/aws-sdk-go from 1.43.17 to 1.43.26 [#2018](https://github.com/GoogleContainerTools/kaniko/pull/2018)
* Bump github.com/containerd/containerd from 1.6.1 to 1.6.2 [#2017](https://github.com/GoogleContainerTools/kaniko/pull/2017)
* Bump github.com/docker/docker from 20.10.13+incompatible to 20.10.14+incompatible [#2016](https://github.com/GoogleContainerTools/kaniko/pull/2016)
* README.md: Update docs on building for AWS ECR [#2020](https://github.com/GoogleContainerTools/kaniko/pull/2020)
* Move and fix GetContainerRuntime check from bpfd proc [#1996](https://github.com/GoogleContainerTools/kaniko/pull/1996)
* Fix minor glitch in the SVG logos [#2004](https://github.com/GoogleContainerTools/kaniko/pull/2004)
* Add SVG logos [#2002](https://github.com/GoogleContainerTools/kaniko/pull/2002)
* Bump github/codeql-action from 1.1.3 to 1.1.5 [#2000](https://github.com/GoogleContainerTools/kaniko/pull/2000)
* Fix - Incomplete regular expression for hostnames [#1993](https://github.com/GoogleContainerTools/kaniko/pull/1993)
* Bump github.com/spf13/cobra from 1.3.0 to 1.4.0 [#1985](https://github.com/GoogleContainerTools/kaniko/pull/1985)
* Bump github.com/aws/aws-sdk-go from 1.43.12 to 1.43.17 [#1986](https://github.com/GoogleContainerTools/kaniko/pull/1986)
* Bump github.com/spf13/afero from 1.8.1 to 1.8.2 [#1987](https://github.com/GoogleContainerTools/kaniko/pull/1987)
* Bump github.com/docker/docker from 20.10.12+incompatible to 20.10.13+incompatible [#1988](https://github.com/GoogleContainerTools/kaniko/pull/1988)
* Fix image tags in release workflow [#1977](https://github.com/GoogleContainerTools/kaniko/pull/1977)
* Use canonical platform values. Fix 1995. [#2025](https://github.com/GoogleContainerTools/kaniko/pull/2025)
* feat: kaniko dir config option [#1997](https://github.com/GoogleContainerTools/kaniko/pull/1997)
* Bump github.com/aws/aws-sdk-go from 1.43.17 to 1.43.26 [#2018](https://github.com/GoogleContainerTools/kaniko/pull/2018)
* Bump github.com/containerd/containerd from 1.6.1 to 1.6.2 [#2017](https://github.com/GoogleContainerTools/kaniko/pull/2017)
* Bump github.com/docker/docker from 20.10.13+incompatible to 20.10.14+incompatible [#2016](https://github.com/GoogleContainerTools/kaniko/pull/2016)
* README.md: Update docs on building for AWS ECR [#2020](https://github.com/GoogleContainerTools/kaniko/pull/2020)
* Move and fix GetContainerRuntime check from bpfd proc [#1996](https://github.com/GoogleContainerTools/kaniko/pull/1996)
* Fix minor glitch in the SVG logos [#2004](https://github.com/GoogleContainerTools/kaniko/pull/2004)
* Add SVG logos [#2002](https://github.com/GoogleContainerTools/kaniko/pull/2002)
* Bump github/codeql-action from 1.1.3 to 1.1.5 [#2000](https://github.com/GoogleContainerTools/kaniko/pull/2000)
* Fix - Incomplete regular expression for hostnames [#1993](https://github.com/GoogleContainerTools/kaniko/pull/1993)
* Bump github.com/spf13/cobra from 1.3.0 to 1.4.0 [#1985](https://github.com/GoogleContainerTools/kaniko/pull/1985)
* Bump github.com/aws/aws-sdk-go from 1.43.12 to 1.43.17 [#1986](https://github.com/GoogleContainerTools/kaniko/pull/1986)
* Bump github.com/spf13/afero from 1.8.1 to 1.8.2 [#1987](https://github.com/GoogleContainerTools/kaniko/pull/1987)
* Bump github.com/docker/docker from 20.10.12+incompatible to 20.10.13+incompatible [#1988](https://github.com/GoogleContainerTools/kaniko/pull/1988)
* Fix image tags in release workflow [#1977](https://github.com/GoogleContainerTools/kaniko/pull/1977)
Huge thank you for this release towards our contributors:
- Aaruni Aggarwal
- Adrian Newby
- Anbraten
- Andrei Kvapil
- ankitm123
- Benjamin Krenn
- Bernardo Marques
- Chuang Wang
- Dávid Szakállas
- Dawei Ma
- dependabot[bot]
- Diego Gonzalez
- ejose19
- Eng Zer Jun
- Florian Apolloner
- François JACQUES
- Gabriel Nützi
- Gilbert Gilb's
- Guillaume Calmettes
- Herman
- Igor Scheller
- Jack
- Jake Sanders
- Janosch Maier
- Jason Hall
- Jasper Ben Orschulko
- jeunii
- Jose Donizetti
- Kamal Nasser
- Kun Lu
- Lars Seipel
- Liwen Guo
- Matt Moore
- Max Walther
- Mikhail Vasin
- Naveen
- nihilo
- Oliver Gregorius
- Pat Litke
- Patrick Barker
- priyawadhwa
- Rhianna
- Sebastiaan Tammer
- Sigurd Spieckermann
- Silvano Cirujano Cuesta
- Tejal Desai
- Travis DePrato
- Wolfgang Walther
- wwade
- Yahav Itzhak
- ygelfand


# v1.8.0 Release 2022-03-08
This is Mar's 2022 release.

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.8.0
gcr.io/kaniko-project/executor:latest
```

The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.8.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.8.0-slim
```

* Update dependabot settings to get updates for docker [#1969](https://github.com/GoogleContainerTools/kaniko/pull/1969)
* Bump actions/setup-go from 2.2.0 to 3 [#1970](https://github.com/GoogleContainerTools/kaniko/pull/1970)
* Bump google-github-actions/setup-gcloud from 0.5.0 to 0.5.1 [#1950](https://github.com/GoogleContainerTools/kaniko/pull/1950)
* Pinned GitHub actions by SHA [#1963](https://github.com/GoogleContainerTools/kaniko/pull/1963)
* Bump actions/upload-artifact from 2.3.1 to 3 [#1968](https://github.com/GoogleContainerTools/kaniko/pull/1968)
* Bump actions/checkout from 2 to 3 [#1967](https://github.com/GoogleContainerTools/kaniko/pull/1967)
* Bump github.com/aws/aws-sdk-go from 1.42.52 to 1.43.12 [#1966](https://github.com/GoogleContainerTools/kaniko/pull/1966)
* Bump github.com/containerd/containerd from 1.6.0 to 1.6.1 [#1961](https://github.com/GoogleContainerTools/kaniko/pull/1961)
* Fix bug with log disabling [#1959](https://github.com/GoogleContainerTools/kaniko/pull/1959)
* Bump github/codeql-action from 1.1.2 to 1.1.3 [#1958](https://github.com/GoogleContainerTools/kaniko/pull/1958)
* Bump github.com/aws/aws-sdk-go from 1.42.52 to 1.43.7 [#1957](https://github.com/GoogleContainerTools/kaniko/pull/1957)
* Removed --whitelist-var-run normalization as this breaks functionality [#1956](https://github.com/GoogleContainerTools/kaniko/pull/1956)
* Bump github.com/containerd/containerd from 1.5.9 to 1.6.0 [#1948](https://github.com/GoogleContainerTools/kaniko/pull/1948)
* Bump cloud.google.com/go/storage from 1.20.0 to 1.21.0 [#1947](https://github.com/GoogleContainerTools/kaniko/pull/1947)
* Bump github/codeql-action from 1.1.0 to 1.1.2 [#1951](https://github.com/GoogleContainerTools/kaniko/pull/1951)
* Bump ossf/scorecard-action from 1.0.3 to 1.0.4 [#1952](https://github.com/GoogleContainerTools/kaniko/pull/1952)
* Bump ecr-login dep to avoid some log spam [#1946](https://github.com/GoogleContainerTools/kaniko/pull/1946)
* readme: Fix formatting for `--image-fs-extract-retry` [#1942](https://github.com/GoogleContainerTools/kaniko/pull/1942)
* Pick up per-repository auth changes from go-containerregistry [#1939](https://github.com/GoogleContainerTools/kaniko/pull/1939)
* Bump github.com/aws/aws-sdk-go from 1.42.47 to 1.42.52 [#1937](https://github.com/GoogleContainerTools/kaniko/pull/1937)
* Bump github/codeql-action from 1.0.31 to 1.1.0 [#1938](https://github.com/GoogleContainerTools/kaniko/pull/1938)
* Set DOCKER_BUILDKIT=1 in make images [#1906](https://github.com/GoogleContainerTools/kaniko/pull/1906)
* Fix resolving arguments over multi-stage build [#1928](https://github.com/GoogleContainerTools/kaniko/pull/1928)
* Correctly handle platforms that include CPU variants [#1929](https://github.com/GoogleContainerTools/kaniko/pull/1929)
* Restore build args after optimize. Fixes #1910, #1912. [#1915](https://github.com/GoogleContainerTools/kaniko/pull/1915)
* test: use `T.TempDir` to create temporary test directory [#1918](https://github.com/GoogleContainerTools/kaniko/pull/1918)
* Bump github.com/spf13/afero from 1.8.0 to 1.8.1 [#1922](https://github.com/GoogleContainerTools/kaniko/pull/1922)
* Bump github.com/aws/aws-sdk-go from 1.42.44 to 1.42.47 [#1923](https://github.com/GoogleContainerTools/kaniko/pull/1923)
* Bump cloud.google.com/go/storage from 1.19.0 to 1.20.0 [#1924](https://github.com/GoogleContainerTools/kaniko/pull/1924)
* Bump ossf/scorecard-action from 1.0.2 to 1.0.3 [#1926](https://github.com/GoogleContainerTools/kaniko/pull/1926)
* Bump google-github-actions/setup-gcloud from 0.4.0 to 0.5.0 [#1925](https://github.com/GoogleContainerTools/kaniko/pull/1925)
* Bump github/codeql-action from 1.0.30 to 1.0.31 [#1927](https://github.com/GoogleContainerTools/kaniko/pull/1927)
* Vagrantfile should install and configure go (see #1913) [#1914](https://github.com/GoogleContainerTools/kaniko/pull/1914)
* adding ppc64le support for executor and warmer image [#1908](https://github.com/GoogleContainerTools/kaniko/pull/1908)
* Remove deploy/cloudbuild-*.yaml files [#1907](https://github.com/GoogleContainerTools/kaniko/pull/1907)
* Bump go-containerregistry to pick up ACR fix [#1898](https://github.com/GoogleContainerTools/kaniko/pull/1898)
* Bump cloud.google.com/go/storage from 1.18.2 to 1.19.0 [#1903](https://github.com/GoogleContainerTools/kaniko/pull/1903)
* Bump github.com/aws/aws-sdk-go from 1.42.38 to 1.42.44 [#1902](https://github.com/GoogleContainerTools/kaniko/pull/1902)
* Bump ossf/scorecard-action from 5da1b6b2680a229f2e66131f5c6a692bcd80b246 to 1.0.2 [#1899](https://github.com/GoogleContainerTools/kaniko/pull/1899)
* Bump google-github-actions/setup-gcloud from 0.3.0 to 0.4.0 [#1900](https://github.com/GoogleContainerTools/kaniko/pull/1900)
* Bump github/codeql-action from 1.0.26 to 1.0.30 [#1901](https://github.com/GoogleContainerTools/kaniko/pull/1901)
* Enable dependabot for Go and GitHub Actions dependencies [#1884](https://github.com/GoogleContainerTools/kaniko/pull/1884)
* Update readme [#1897](https://github.com/GoogleContainerTools/kaniko/pull/1897)
* Remove k8schain, directly depend on cred helpers [#1891](https://github.com/GoogleContainerTools/kaniko/pull/1891)
* Update golang.org/x/oauth2/google [#1890](https://github.com/GoogleContainerTools/kaniko/pull/1890)
* Bump dependencies [#1885](https://github.com/GoogleContainerTools/kaniko/pull/1885)
* Fix broken anchor link [#1804](https://github.com/GoogleContainerTools/kaniko/pull/1804)
* Bump github.com/docker/docker to latest release [#1866](https://github.com/GoogleContainerTools/kaniko/pull/1866)
* Run GitHub Actions on pushes and PRs to main, not master [#1883](https://github.com/GoogleContainerTools/kaniko/pull/1883)
* Add KANIKO_REGISTRY_MIRROR env var [#1875](https://github.com/GoogleContainerTools/kaniko/pull/1875)
* Bump AWS ecr-login cred helper to v0.5.0 [#1880](https://github.com/GoogleContainerTools/kaniko/pull/1880)
* Pin to more recent version of scorecard [#1878](https://github.com/GoogleContainerTools/kaniko/pull/1878)
* Add ossf/scorecard Github Action to kaniko [#1874](https://github.com/GoogleContainerTools/kaniko/pull/1874)
* Attempt to fix erroneous build cancellation [#1867](https://github.com/GoogleContainerTools/kaniko/pull/1867)
* Add s390x support to docker images [#1749](https://github.com/GoogleContainerTools/kaniko/pull/1749)
* fix: ARG/ENV used in script does not invalidate build cache (#1688) [#1693](https://github.com/GoogleContainerTools/kaniko/pull/1693)
* fix: change the name of the acr cred helper [#1865](https://github.com/GoogleContainerTools/kaniko/pull/1865)
* Fix implicit GCR auth [#1856](https://github.com/GoogleContainerTools/kaniko/pull/1856)
* Log full image ref by digest when pushing an image [#1857](https://github.com/GoogleContainerTools/kaniko/pull/1857)
* Remove GitHub Actions concurrency limits [#1858](https://github.com/GoogleContainerTools/kaniko/pull/1858)
* tar: read directly from stdin [#1728](https://github.com/GoogleContainerTools/kaniko/pull/1728)
* Fix regression: can fetch branches and tags references without specifying commit hashes for private git repository used as context [#1823](https://github.com/GoogleContainerTools/kaniko/pull/1823)
* Use pax tar format [#1809](https://github.com/GoogleContainerTools/kaniko/pull/1809)
* Fix calculating path for copying ownership [#1859](https://github.com/GoogleContainerTools/kaniko/pull/1859)
* Fix copying ownership [#1725](https://github.com/GoogleContainerTools/kaniko/pull/1725)
* Fix typo [#1825](https://github.com/GoogleContainerTools/kaniko/pull/1825)
* Fix possible nil pointer derefence in fs_util.go [#1813](https://github.com/GoogleContainerTools/kaniko/pull/1813)
* include auth for FetchOptions [#1796](https://github.com/GoogleContainerTools/kaniko/pull/1796)
* Update readme insecure flags [#1811](https://github.com/GoogleContainerTools/kaniko/pull/1811)
* Add documentation on pushing to ACR [#1831](https://github.com/GoogleContainerTools/kaniko/pull/1831)
* Fixes #1837 : keep file capabilities on archival [#1838](https://github.com/GoogleContainerTools/kaniko/pull/1838)
* Use setup-gcloud@v0.3.0 instead of @master [#1854](https://github.com/GoogleContainerTools/kaniko/pull/1854)
* Collapse integration test workflows into one config [#1855](https://github.com/GoogleContainerTools/kaniko/pull/1855)
* Share the Go build cache when building in Dockerfiles [#1853](https://github.com/GoogleContainerTools/kaniko/pull/1853)
* Call cosign sign --key [#1849](https://github.com/GoogleContainerTools/kaniko/pull/1849)
* Consolidate PR and real release workflows [#1845](https://github.com/GoogleContainerTools/kaniko/pull/1845)
* Use golang:1.17 and build from reproducible source [#1848](https://github.com/GoogleContainerTools/kaniko/pull/1848)
* Start keyless signing kaniko releases [#1841](https://github.com/GoogleContainerTools/kaniko/pull/1841)
* Attempt to speed up PR image builds by sharing a cache [#1844](https://github.com/GoogleContainerTools/kaniko/pull/1844)
* Sign digests not tags. [#1840](https://github.com/GoogleContainerTools/kaniko/pull/1840)
* Fix the e2e K8s test [#1842](https://github.com/GoogleContainerTools/kaniko/pull/1842)
* Bump the cosign version (a lot) [#1839](https://github.com/GoogleContainerTools/kaniko/pull/1839)
* Revert "Support mirror registries with path component (#1707)" [#1794](https://github.com/GoogleContainerTools/kaniko/pull/1794)
* Fix syntax error in release.yaml [#1800](https://github.com/GoogleContainerTools/kaniko/pull/1800)
* Update dependabot settings to get updates for docker [#1969](https://github.com/GoogleContainerTools/kaniko/pull/1969)
* Bump actions/setup-go from 2.2.0 to 3 [#1970](https://github.com/GoogleContainerTools/kaniko/pull/1970)
* Bump google-github-actions/setup-gcloud from 0.5.0 to 0.5.1 [#1950](https://github.com/GoogleContainerTools/kaniko/pull/1950)
* Pinned GitHub actions by SHA [#1963](https://github.com/GoogleContainerTools/kaniko/pull/1963)
* Bump actions/upload-artifact from 2.3.1 to 3 [#1968](https://github.com/GoogleContainerTools/kaniko/pull/1968)
* Bump actions/checkout from 2 to 3 [#1967](https://github.com/GoogleContainerTools/kaniko/pull/1967)
* Bump github.com/aws/aws-sdk-go from 1.42.52 to 1.43.12 [#1966](https://github.com/GoogleContainerTools/kaniko/pull/1966)
* Bump github.com/containerd/containerd from 1.6.0 to 1.6.1 [#1961](https://github.com/GoogleContainerTools/kaniko/pull/1961)
* Fix bug with log disabling [#1959](https://github.com/GoogleContainerTools/kaniko/pull/1959)
* Bump github/codeql-action from 1.1.2 to 1.1.3 [#1958](https://github.com/GoogleContainerTools/kaniko/pull/1958)
* Bump github.com/aws/aws-sdk-go from 1.42.52 to 1.43.7 [#1957](https://github.com/GoogleContainerTools/kaniko/pull/1957)
* Removed --whitelist-var-run normalization as this breaks functionality [#1956](https://github.com/GoogleContainerTools/kaniko/pull/1956)
* Bump github.com/containerd/containerd from 1.5.9 to 1.6.0 [#1948](https://github.com/GoogleContainerTools/kaniko/pull/1948)
* Bump cloud.google.com/go/storage from 1.20.0 to 1.21.0 [#1947](https://github.com/GoogleContainerTools/kaniko/pull/1947)
* Bump github/codeql-action from 1.1.0 to 1.1.2 [#1951](https://github.com/GoogleContainerTools/kaniko/pull/1951)
* Bump ossf/scorecard-action from 1.0.3 to 1.0.4 [#1952](https://github.com/GoogleContainerTools/kaniko/pull/1952)
* Bump ecr-login dep to avoid some log spam [#1946](https://github.com/GoogleContainerTools/kaniko/pull/1946)
* readme: Fix formatting for `--image-fs-extract-retry` [#1942](https://github.com/GoogleContainerTools/kaniko/pull/1942)
* Pick up per-repository auth changes from go-containerregistry [#1939](https://github.com/GoogleContainerTools/kaniko/pull/1939)
* Bump github.com/aws/aws-sdk-go from 1.42.47 to 1.42.52 [#1937](https://github.com/GoogleContainerTools/kaniko/pull/1937)
* Bump github/codeql-action from 1.0.31 to 1.1.0 [#1938](https://github.com/GoogleContainerTools/kaniko/pull/1938)
* Set DOCKER_BUILDKIT=1 in make images [#1906](https://github.com/GoogleContainerTools/kaniko/pull/1906)
* Fix resolving arguments over multi-stage build [#1928](https://github.com/GoogleContainerTools/kaniko/pull/1928)
* Correctly handle platforms that include CPU variants [#1929](https://github.com/GoogleContainerTools/kaniko/pull/1929)
* Restore build args after optimize. Fixes #1910, #1912. [#1915](https://github.com/GoogleContainerTools/kaniko/pull/1915)
* test: use `T.TempDir` to create temporary test directory [#1918](https://github.com/GoogleContainerTools/kaniko/pull/1918)
* Bump github.com/spf13/afero from 1.8.0 to 1.8.1 [#1922](https://github.com/GoogleContainerTools/kaniko/pull/1922)
* Bump github.com/aws/aws-sdk-go from 1.42.44 to 1.42.47 [#1923](https://github.com/GoogleContainerTools/kaniko/pull/1923)
* Bump cloud.google.com/go/storage from 1.19.0 to 1.20.0 [#1924](https://github.com/GoogleContainerTools/kaniko/pull/1924)
* Bump ossf/scorecard-action from 1.0.2 to 1.0.3 [#1926](https://github.com/GoogleContainerTools/kaniko/pull/1926)
* Bump google-github-actions/setup-gcloud from 0.4.0 to 0.5.0 [#1925](https://github.com/GoogleContainerTools/kaniko/pull/1925)
* Bump github/codeql-action from 1.0.30 to 1.0.31 [#1927](https://github.com/GoogleContainerTools/kaniko/pull/1927)
* Vagrantfile should install and configure go (see #1913) [#1914](https://github.com/GoogleContainerTools/kaniko/pull/1914)
* adding ppc64le support for executor and warmer image [#1908](https://github.com/GoogleContainerTools/kaniko/pull/1908)
* Remove deploy/cloudbuild-*.yaml files [#1907](https://github.com/GoogleContainerTools/kaniko/pull/1907)
* Bump go-containerregistry to pick up ACR fix [#1898](https://github.com/GoogleContainerTools/kaniko/pull/1898)
* Bump cloud.google.com/go/storage from 1.18.2 to 1.19.0 [#1903](https://github.com/GoogleContainerTools/kaniko/pull/1903)
* Bump github.com/aws/aws-sdk-go from 1.42.38 to 1.42.44 [#1902](https://github.com/GoogleContainerTools/kaniko/pull/1902)
* Bump ossf/scorecard-action from 5da1b6b2680a229f2e66131f5c6a692bcd80b246 to 1.0.2 [#1899](https://github.com/GoogleContainerTools/kaniko/pull/1899)
* Bump google-github-actions/setup-gcloud from 0.3.0 to 0.4.0 [#1900](https://github.com/GoogleContainerTools/kaniko/pull/1900)
* Bump github/codeql-action from 1.0.26 to 1.0.30 [#1901](https://github.com/GoogleContainerTools/kaniko/pull/1901)
* Enable dependabot for Go and GitHub Actions dependencies [#1884](https://github.com/GoogleContainerTools/kaniko/pull/1884)
* Update readme [#1897](https://github.com/GoogleContainerTools/kaniko/pull/1897)
* Remove k8schain, directly depend on cred helpers [#1891](https://github.com/GoogleContainerTools/kaniko/pull/1891)
* Update golang.org/x/oauth2/google [#1890](https://github.com/GoogleContainerTools/kaniko/pull/1890)
* Bump dependencies [#1885](https://github.com/GoogleContainerTools/kaniko/pull/1885)
* Fix broken anchor link [#1804](https://github.com/GoogleContainerTools/kaniko/pull/1804)
* Bump github.com/docker/docker to latest release [#1866](https://github.com/GoogleContainerTools/kaniko/pull/1866)
* Run GitHub Actions on pushes and PRs to main, not master [#1883](https://github.com/GoogleContainerTools/kaniko/pull/1883)
* Add KANIKO_REGISTRY_MIRROR env var [#1875](https://github.com/GoogleContainerTools/kaniko/pull/1875)
* Bump AWS ecr-login cred helper to v0.5.0 [#1880](https://github.com/GoogleContainerTools/kaniko/pull/1880)
* Pin to more recent version of scorecard [#1878](https://github.com/GoogleContainerTools/kaniko/pull/1878)
* Add ossf/scorecard Github Action to kaniko [#1874](https://github.com/GoogleContainerTools/kaniko/pull/1874)
* Attempt to fix erroneous build cancellation [#1867](https://github.com/GoogleContainerTools/kaniko/pull/1867)
* Add s390x support to docker images [#1749](https://github.com/GoogleContainerTools/kaniko/pull/1749)
* fix: ARG/ENV used in script does not invalidate build cache (#1688) [#1693](https://github.com/GoogleContainerTools/kaniko/pull/1693)
* fix: change the name of the acr cred helper [#1865](https://github.com/GoogleContainerTools/kaniko/pull/1865)
* Fix implicit GCR auth [#1856](https://github.com/GoogleContainerTools/kaniko/pull/1856)
* Log full image ref by digest when pushing an image [#1857](https://github.com/GoogleContainerTools/kaniko/pull/1857)
* Remove GitHub Actions concurrency limits [#1858](https://github.com/GoogleContainerTools/kaniko/pull/1858)
* tar: read directly from stdin [#1728](https://github.com/GoogleContainerTools/kaniko/pull/1728)
* Fix regression: can fetch branches and tags references without specifying commit hashes for private git repository used as context [#1823](https://github.com/GoogleContainerTools/kaniko/pull/1823)
* Use pax tar format [#1809](https://github.com/GoogleContainerTools/kaniko/pull/1809)
* Fix calculating path for copying ownership [#1859](https://github.com/GoogleContainerTools/kaniko/pull/1859)
* Fix copying ownership [#1725](https://github.com/GoogleContainerTools/kaniko/pull/1725)
* Fix typo [#1825](https://github.com/GoogleContainerTools/kaniko/pull/1825)
* Fix possible nil pointer derefence in fs_util.go [#1813](https://github.com/GoogleContainerTools/kaniko/pull/1813)
* include auth for FetchOptions [#1796](https://github.com/GoogleContainerTools/kaniko/pull/1796)
* Update readme insecure flags [#1811](https://github.com/GoogleContainerTools/kaniko/pull/1811)
* Add documentation on pushing to ACR [#1831](https://github.com/GoogleContainerTools/kaniko/pull/1831)
* Fixes #1837 : keep file capabilities on archival [#1838](https://github.com/GoogleContainerTools/kaniko/pull/1838)
* Use setup-gcloud@v0.3.0 instead of @master [#1854](https://github.com/GoogleContainerTools/kaniko/pull/1854)
* Collapse integration test workflows into one config [#1855](https://github.com/GoogleContainerTools/kaniko/pull/1855)
* Share the Go build cache when building in Dockerfiles [#1853](https://github.com/GoogleContainerTools/kaniko/pull/1853)
* Call cosign sign --key [#1849](https://github.com/GoogleContainerTools/kaniko/pull/1849)
* Consolidate PR and real release workflows [#1845](https://github.com/GoogleContainerTools/kaniko/pull/1845)
* Use golang:1.17 and build from reproducible source [#1848](https://github.com/GoogleContainerTools/kaniko/pull/1848)
* Start keyless signing kaniko releases [#1841](https://github.com/GoogleContainerTools/kaniko/pull/1841)
* Attempt to speed up PR image builds by sharing a cache [#1844](https://github.com/GoogleContainerTools/kaniko/pull/1844)
* Sign digests not tags. [#1840](https://github.com/GoogleContainerTools/kaniko/pull/1840)
* Fix the e2e K8s test [#1842](https://github.com/GoogleContainerTools/kaniko/pull/1842)
* Bump the cosign version (a lot) [#1839](https://github.com/GoogleContainerTools/kaniko/pull/1839)
* Revert "Support mirror registries with path component (#1707)" [#1794](https://github.com/GoogleContainerTools/kaniko/pull/1794)
* Fix syntax error in release.yaml [#1800](https://github.com/GoogleContainerTools/kaniko/pull/1800)
Huge thank you for this release towards our contributors:
- Aaruni Aggarwal
- Adrian Newby
- Anbraten
- Andrei Kvapil
- ankitm123
- Benjamin Krenn
- Bernardo Marques
- Dávid Szakállas
- Dawei Ma
- dependabot[bot]
- ejose19
- Eng Zer Jun
- Florian Apolloner
- François JACQUES
- Gabriel Nützi
- Gilbert Gilb's
- Guillaume Calmettes
- Herman
- Jake Sanders
- Janosch Maier
- Jason Hall
- jeunii
- Jose Donizetti
- Kamal Nasser
- Kun Lu
- Lars Seipel
- Liwen Guo
- Matt Moore
- Max Walther
- Mikhail Vasin
- Naveen
- nihilo
- Oliver Gregorius
- Pat Litke
- Patrick Barker
- priyawadhwa
- Rhianna
- Sebastiaan Tammer
- Silvano Cirujano Cuesta
- Tejal Desai
- Travis DePrato
- Wolfgang Walther
- wwade
- Yahav Itzhak
- ygelfand


# v1.7.0 Release 2021-10-19
This is Oct's 2021 release.

## Highights

* In this release, we have kaniko **s390x** platform support for multi-arch image.
* Kaniko **Self Serve** documentation is up to enableuser to build and push kaniko images themselves [here](https://github.com/GoogleContainerTools/kaniko/blob/master/RELEASE.md)



The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.7.0
gcr.io/kaniko-project/executor:latest
```
The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.7.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.7.0-slim
```

*  git: accept explicit commit hash for git context [#1765](https://github.com/GoogleContainerTools/kaniko/pull/1765)
* Remove tarball.WithCompressedCaching flag to resolve OOM Killed error [#1722](https://github.com/GoogleContainerTools/kaniko/pull/1722)
* disable github action workflow on push to master [#1770](https://github.com/GoogleContainerTools/kaniko/pull/1770)
* Add s390x support to docker images [copy] [#1769](https://github.com/GoogleContainerTools/kaniko/pull/1769)
* Fix typo [#1719](https://github.com/GoogleContainerTools/kaniko/pull/1719)
* Fix composite cache key for multi-stage copy command [#1735](https://github.com/GoogleContainerTools/kaniko/pull/1735)
* chore: add workflows for pr tests [#1766](https://github.com/GoogleContainerTools/kaniko/pull/1766)
* Make /bin/sh available to debug image [#1748](https://github.com/GoogleContainerTools/kaniko/pull/1748)
* Fix executor Dockerfile, which wasn't building [#1741](https://github.com/GoogleContainerTools/kaniko/pull/1741)
* Support force-building metadata layers into snapshot [#1731](https://github.com/GoogleContainerTools/kaniko/pull/1731)
* Add support for CPU variants [#1676](https://github.com/GoogleContainerTools/kaniko/pull/1676)
* refactor: adjust bpfd container runtime detection [#1686](https://github.com/GoogleContainerTools/kaniko/pull/1686)
* Fix snapshotter ignore list; do not attempt to delete whiteouts of ignored paths [#1652](https://github.com/GoogleContainerTools/kaniko/pull/1652)
* Add instructions for using JFrog Artifactory [#1715](https://github.com/GoogleContainerTools/kaniko/pull/1715)
* add SECURITY.md [#1710](https://github.com/GoogleContainerTools/kaniko/pull/1710)
* Support mirror registries with path component [#1707](https://github.com/GoogleContainerTools/kaniko/pull/1707)
* Retry extracting filesystem from image [#1685](https://github.com/GoogleContainerTools/kaniko/pull/1685)
* Bugfix/trailing path separator [#1683](https://github.com/GoogleContainerTools/kaniko/pull/1683)
* docs: add missing cache-copy-layers arg in README [#1672](https://github.com/GoogleContainerTools/kaniko/pull/1672)
* save snaphots to tmp dir [#1662](https://github.com/GoogleContainerTools/kaniko/pull/1662)
* Revert "save snaphots to tmp dir" [#1670](https://github.com/GoogleContainerTools/kaniko/pull/1670)
* Try to warm all images and warn about errors [#1653](https://github.com/GoogleContainerTools/kaniko/pull/1653)
* Exit Code Propagation [#1655](https://github.com/GoogleContainerTools/kaniko/pull/1655)
* Fix changelog headings [#1643](https://github.com/GoogleContainerTools/kaniko/pull/1643)


Huge thank you for this release towards our contributors:
- Anbraten
- Benjamin Krenn
- Gilbert Gilb's
- Jake Sanders
- Janosch Maier
- Jason Hall
- Jose Donizetti
- Kamal Nasser
- Liwen Guo
- Max Walther
- Mikhail Vasin
- Patrick Barker
- Rhianna
- Silvano Cirujano Cuesta
- Tejal Desai
- Yahav Itzhak
- ankitm123
- ejose19
- nihilo
- priyawadhwa
- wwade

# v1.6.0 Release 2021-04-23
This is April's 2021 release.

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.6.0
gcr.io/kaniko-project/executor:latest
```
The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.6.0-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.6.0-slim
```

* Support warming images by digest [#1629](https://github.com/GoogleContainerTools/kaniko/pull/1629)
* Fix resolution of Dockerfile relative dockerignore [#1607](https://github.com/GoogleContainerTools/kaniko/pull/1607)
* create parent directory before writing digest files [#1612](https://github.com/GoogleContainerTools/kaniko/pull/1612)
* adds ignore-path command arguments to executor [#1622](https://github.com/GoogleContainerTools/kaniko/pull/1622)
* Specifying a tarPath will push the image as well [#1597](https://github.com/GoogleContainerTools/kaniko/pull/1597)

Huge thank you for this release towards our contributors:
- Chris Hoffman
- Colin
- Jon Friesen
- Lars Gröber
- Sascha Schwarze
- Tejal Desai
- Viktor Farcic
- Vivek Kumar
- priyawadhwa

# v1.5.2 Release 2021-03-30

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.5.2
gcr.io/kaniko-project/executor:latest
```
The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.5.2-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.5.2-slim
```

This release is the first to be signed by [cosign](https://github.com/sigstore/cosign)!
The PEM-encoded public key to validate against the released kaniko images is:

```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE9aAfAcgAxIFMTstJUv8l/AMqnSKw
P+vLu3NnnBDHCfREQpV/AJuiZ1UtgGpFpHlJLCNPmFkzQTnfyN5idzNl6Q==
-----END PUBLIC KEY-----
```

# v1.5.1 Release 2021-02-22
This release is a minor release with following a fix to version number for v1.5.0
The kaniko images now report the right version number.

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.5.1
gcr.io/kaniko-project/executor:latest
```
The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.5.1-debug
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:v1.5.1-slim
```

In this release, we have 1 new feature:
* Improve retry behavior for push operation [#1578](https://github.com/GoogleContainerTools/kaniko/pull/1578)

And followinf refactors/updates to documentation
* Added a video introduction to Kaniko [#1517](https://github.com/GoogleContainerTools/kaniko/pull/1517)
* Use up-to-date ca-certificates during build [#1580](https://github.com/GoogleContainerTools/kaniko/pull/1580)


Huge thank you for this release towards our contributors:
- Sascha Schwarze
- Tejal Desai
- Viktor Farcic

# v1.5.0 Release 2021-01-25

This releases publishes multi-arch image kaniko images for following platforms
1. linux/amd64
2. linux/arm64
3. linux/ppc64le

If you want to add other platforms, please talk to @tejal29.

The executor images in this release are:
```
gcr.io/kaniko-project/executor:v1.5.0
gcr.io/kaniko-project/executor:latest
```
The debug images are available at:
```
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:v1.5.0-debug
```

In this release, we have 2 slim executor images which don't contain any authentication binaries.

1. `gcr.io/kaniko-project/executor:slim`  &
2. `gcr.io/kaniko-project/executor:v1.5.0-slim`


## New Features

* Mutli-arch support  [#1531](https://github.com/GoogleContainerTools/kaniko/pull/1531), [#1474](https://github.com/GoogleContainerTools/kaniko/pull/1474)
* Add support to fetch a github pull request [#1543](https://github.com/GoogleContainerTools/kaniko/pull/1543)
* Add --image-name-tag-with-digest flag [#1541](https://github.com/GoogleContainerTools/kaniko/pull/1541)
* add caching copy layers back [#1518](https://github.com/GoogleContainerTools/kaniko/pull/1518)
* Profiling for Snapshotting errors. [#1530](https://github.com/GoogleContainerTools/kaniko/pull/1530)
* feat(warmer): Warmer now supports all registry-related flags [#1499](https://github.com/GoogleContainerTools/kaniko/pull/1499)
* feat: Add https tar.gz remote source for context [#1519](https://github.com/GoogleContainerTools/kaniko/pull/1519)
* Add option customPlatform [#1500](https://github.com/GoogleContainerTools/kaniko/pull/1500)
* feat: support multiple registry mirrors with fallback [#1498](https://github.com/GoogleContainerTools/kaniko/pull/1498)
* Add s390x kaniko build to multi-arch list [#1475](https://github.com/GoogleContainerTools/kaniko/pull/1475)

## Bug Fixes
* reject tarball writes with no destinations [#1534](https://github.com/GoogleContainerTools/kaniko/pull/1534)
* Fix travis-ci link [#1535](https://github.com/GoogleContainerTools/kaniko/pull/1535)
* fix: extract file as same user for warmer docker image [#1538](https://github.com/GoogleContainerTools/kaniko/pull/1538)
* fix: update busybox version to fix CVE-2018-1000500 [#1532](https://github.com/GoogleContainerTools/kaniko/pull/1532)
* Fix typo in error message [#1494](https://github.com/GoogleContainerTools/kaniko/pull/1494)
* Fix COPY with --chown command [#1477](https://github.com/GoogleContainerTools/kaniko/pull/1477)
* Remove unused code [#1495](https://github.com/GoogleContainerTools/kaniko/pull/1495)
* Fixes #1469 : Remove file that matches with the directory path [#1478](https://github.com/GoogleContainerTools/kaniko/pull/1478)
* fix: CheckPushPermissions not being called when using --no-push and --cache-repo [#1471](https://github.com/GoogleContainerTools/kaniko/pull/1471)

## Refactors
* Switch to runtime detection via bpfd/proc [#1502](https://github.com/GoogleContainerTools/kaniko/pull/1502)
* Update ggcr to pick up estargz and caching option [#1527](https://github.com/GoogleContainerTools/kaniko/pull/1527)

## Documentation
* Document flags for tarball build only [#1503](https://github.com/GoogleContainerTools/kaniko/pull/1503)
* doc: clarify the format of --registry-mirror [#1504](https://github.com/GoogleContainerTools/kaniko/pull/1504)
* add section to run lints [#1480](https://github.com/GoogleContainerTools/kaniko/pull/1480)
* Add docs for GKE workload identity. [#1476](https://github.com/GoogleContainerTools/kaniko/pull/1476)

Huge thank you for this release towards our contributors:
- Alec Rajeev
- Fabrice
- Josh Chorlton
- Lars
- Lars Toenning
- Matt Moore
- Or Geva
- Severin Strobl
- Shashank
- Sladyn
- Tejal Desai
- Theofilos Papapanagiotou
- Vincent Behar
- Yulia Gaponenko
- ankitm123
- bahetiamit
- ejose19
- mickkael
- zhouhaibing089

# v1.3.0 Release 2020-10-22

This release publishes, multi-arch image kaniko executor images.

Note: The muti-arch images are **only** available for executor images. Contributions Welcome!!

The executor images in this release are:

```
gcr.io/kaniko-project/executor:v1.3.0
gcr.io/kaniko-project/executor:latest

gcr.io/kaniko-project/executor:arm64
gcr.io/kaniko-project/executor:arm64-v1.3.0

gcr.io/kaniko-project/executor:amd64
gcr.io/kaniko-project/executor:amd64-v1.3.0

gcr.io/kaniko-project/executor:multi-arch
gcr.io/kaniko-project/executor:multi-arch-v1.3.0

```
The debug images are available at:
```
gcr.io/kaniko-project/executor:v1.3.0-debug
gcr.io/kaniko-project/executor:debug-v1.3.0 and
gcr.io/kaniko-project/executor:debug
```

## New Features
* Added in docker cred helper for Azure Container Registry sourcing auth tokens directly from environment to debug image [#1458](https://github.com/GoogleContainerTools/kaniko/pull/1458)
* Add multi-arch image via Bazel [#1452](https://github.com/GoogleContainerTools/kaniko/pull/1452)

## Bug Fixes
* Fix docker build tag [#1460](https://github.com/GoogleContainerTools/kaniko/pull/1460)
* Fix .dockerignore for build context copies in later stages [#1447](https://github.com/GoogleContainerTools/kaniko/pull/1447)
* Fix permissions on cache when --no-push is set [#1445](https://github.com/GoogleContainerTools/kaniko/pull/1445)


Huge thank you for this release towards our contributors:

- Akram Ben Aissi
- Alex Szakaly
- Alexander Sharov
- Anthony Davies
- Art Begolli
- Batuhan Apaydın
- Ben Einaudi
- Carlos Alexandro Becker
- Carlos Sanchez
- Chris Mellard
- Chris Sng
- Christopher Hlubek
- Cole Wippern
- Dani Raznikov
- Daniel Marks
- David Dooling
- Didier Durand
- DracoBlue
- Gabriel Virga
- Gilbert Gilb's
- Giovan Isa Musthofa
- Gábor Lipták
- Harmen Stoppels
- Ian Kerins
- James Ravn
- Joe Kutner
- Jon Henrik Bjørnstad
- Jon Johnson
- Jordan GOASDOUE
- Jordan Goasdoue
- Jordan Goasdoué
- Josh Chorlton
- Josh Soref
- Keisuke Umegaki
- Liubov Grinkevich
- Logan.Price
- Lukasz Jakimczuk
- Martin Treusch von Buttlar
- Matt Moore
- Mehdi Abaakouk
- Michel Hollands
- Mitchell Friedman
- Moritz Wanzenböck
- Or Sela
- PhoenixMage
- Pierre-Louis Bonicoli
- Renato Suero
- Sam Stoelinga
- Shihab Hasan
- Sladyn
- Takumasa Sakao
- Tejal Desai
- Thomas Bonfort
- Thomas Stromberg
- Tinjo Schöni
- Tom Prince
- Vincent Latombe
- Wietse Muizelaar
- Yoan Blanc
- Yoriyasu Yano
- Yuheng Zhang
- aca
- cvgw
- ejose19
- ohchang-kwon
- priyawadhwa
- tinkerborg
- tsufeki
- xanonid
- yw-liu
- 好风

# v1.2.0 Release 2020-09-30
This is 27th release of Kaniko!

In this release, Copy layers are not cached there by making builds faster!!
* Stop caching COPY layers [#1408](https://github.com/GoogleContainerTools/kaniko/pull/1408)

Huge thank you for this release towards our contributors:
- Ian Kerins

# v1.1.0 Release 2020-09-30
This is the 26th release of Kaniko!

## New Features
* Add support for Vagrant [#1428](https://github.com/GoogleContainerTools/kaniko/pull/1428)
* Allow DOCKER_CONFIG to be a filename [#1409](https://github.com/GoogleContainerTools/kaniko/pull/1409)

## Bug Fixes
* Fix docker-credential-gcr helper being called for multiple registries [#1439](https://github.com/GoogleContainerTools/kaniko/pull/1439)
* Fix docker-credential-gcr not configured across regions[#1417](https://github.com/GoogleContainerTools/kaniko/pull/1417)

## Updates and Refactors
* add tests for configuring docker credentials across regions. [#1426](https://github.com/GoogleContainerTools/kaniko/pull/1426)

## Documentation
* Update README.md [#1437](https://github.com/GoogleContainerTools/kaniko/pull/1437)
* spelling: storage [#1425](https://github.com/GoogleContainerTools/kaniko/pull/1425)
* Readme.md : Kaniko -> kaniko [#1435](https://github.com/GoogleContainerTools/kaniko/pull/1435)
* initial release instructions [#1419](https://github.com/GoogleContainerTools/kaniko/pull/1419)
* Improve --use-new-run help text, update README with missing flags [#1405](https://github.com/GoogleContainerTools/kaniko/pull/1405)
* Add func to append to ignorelist [#1397](https://github.com/GoogleContainerTools/kaniko/pull/1397)
* Update README.md re: layer cache behavior [#1394](https://github.com/GoogleContainerTools/kaniko/pull/1394)
* Fix links on README [#1398](https://github.com/GoogleContainerTools/kaniko/pull/1398)

Huge thank you for this release towards our contributors:
- aca
- Akram Ben Aissi
- Alexander Sharov
- Alex Szakaly
- Anthony Davies
- Art Begolli
- Batuhan Apaydın
- Ben Einaudi
- Carlos Alexandro Becker
- Carlos Sanchez
- Chris Sng
- Christopher Hlubek
- Cole Wippern
- cvgw
- Daniel Marks
- Dani Raznikov
- David Dooling
- Didier Durand
- DracoBlue
- Gábor Lipták
- Gabriel Virga
- Gilbert Gilb's
- Giovan Isa Musthofa
- Harmen Stoppels
- Ian Kerins
- James Ravn
- Joe Kutner
- Jon Henrik Bjørnstad
- Jon Johnson
- Jordan Goasdoue
- Jordan GOASDOUE
- Jordan Goasdoué
- Josh Chorlton
- Josh Soref
- Keisuke Umegaki
- Liubov Grinkevich
- Logan.Price
- Lukasz Jakimczuk
- Martin Treusch von Buttlar
- Mehdi Abaakouk
- Michel Hollands
- Mitchell Friedman
- Moritz Wanzenböck
- ohchang-kwon
- Or Sela
- PhoenixMage
- Pierre-Louis Bonicoli
- priyawadhwa
- Renato Suero
- Sam Stoelinga
- Shihab Hasan
- Takumasa Sakao
- Tejal Desai
- Thomas Bonfort
- Thomas Stromberg
- Thomas Strömberg
- tinkerborg
- Tom Prince
- tsufeki
- Vincent Latombe
- Wietse Muizelaar
- xanonid
- Yoan Blanc
- Yoriyasu Yano
- Yuheng Zhang
- yw-liu
- 好风

# v1.0.0 Release 2020-08-17
This is the 25th release of Kaniko!

## New Features
* Specify advance options for git checkout branch. [#1322](https://github.com/GoogleContainerTools/kaniko/pull/1322)
  * To specify a branch, use `--git=branch=branchName`
  * To specify an option to checkout a single branch, use `--git=single-branch=true`
  * To change submodule recursions behavior while cloning, use `--git=recurse-submodules=true`
* Checkout a specific git commit [#1153](https://github.com/GoogleContainerTools/kaniko/pull/1153)
* Add ability to specify GIT_TOKEN for git source repository. [#1318](https://github.com/GoogleContainerTools/kaniko/pull/1318)
* The experimental `--use-new-run` flag avoid relying on timestamp. [#1383](https://github.com/GoogleContainerTools/kaniko/pull/1383)

## Bug Fixes
* Set correct PATH for exec form [#1342](https://github.com/GoogleContainerTools/kaniko/pull/1342)
* executor image: fix USER environment variable [#1364](https://github.com/GoogleContainerTools/kaniko/pull/1364)
* fix use new run marker [#1379](https://github.com/GoogleContainerTools/kaniko/pull/1379)
* Use current platform when fetching image in warmer [#1374](https://github.com/GoogleContainerTools/kaniko/pull/1374)
* Bump version number mismatch [#1338](https://github.com/GoogleContainerTools/kaniko/pull/1338)
* Bugfix: Reproducible layers with whiteout [#1350](https://github.com/GoogleContainerTools/kaniko/pull/1350)
* prepend image name when using `registry-mirror` so `library/` is inferred [#1264](https://github.com/GoogleContainerTools/kaniko/pull/1264)
* Add command should fail on 40x when fetching remote file [#1326](https://github.com/GoogleContainerTools/kaniko/pull/1326)

## Refactors & Updates
* bump go-containerregistry dep [#1371](https://github.com/GoogleContainerTools/kaniko/pull/1371)
* feat: upgrade go-git [#1319](https://github.com/GoogleContainerTools/kaniko/pull/1319)
* Move snapshotPathPrefix into a method [#1359](https://github.com/GoogleContainerTools/kaniko/pull/1359)

## Documentation
* Added instructions to use gcr without kubernetes [#1385](https://github.com/GoogleContainerTools/kaniko/pull/1385)
* Format json & yaml in README [#1358](https://github.com/GoogleContainerTools/kaniko/pull/1358)


Huge thank you for this release towards our contributors:
- Alex Szakaly
- Alexander Sharov
- Anthony Davies
- Art Begolli
- Batuhan Apaydın
- Ben Einaudi
- Carlos Alexandro Becker
- Carlos Sanchez
- Chris Sng
- Christopher Hlubek
- Cole Wippern
- Dani Raznikov
- Daniel Marks
- David Dooling
- DracoBlue
- Gabriel Virga
- Gilbert Gilb's
- Giovan Isa Musthofa
- Gábor Lipták
- Harmen Stoppels
- James Ravn
- Joe Kutner
- Jon Henrik Bjørnstad
- Jon Johnson
- Jordan GOASDOUE
- Jordan Goasdoue
- Jordan Goasdoué
- Josh Chorlton
- Liubov Grinkevich
- Logan.Price
- Lukasz Jakimczuk
- Mehdi Abaakouk
- Michel Hollands
- Mitchell Friedman
- Moritz Wanzenböck
- Or Sela
- PhoenixMage
- Pierre-Louis Bonicoli
- Renato Suero
- Sam Stoelinga
- Shihab Hasan
- Takumasa Sakao
- Tejal Desai
- Thomas Bonfort
- Thomas Stromberg
- Thomas Strömberg
- Tom Prince
- Vincent Latombe
- Wietse Muizelaar
- Yoan Blanc
- Yoriyasu Yano
- Yuheng Zhang
- aca
- cvgw
- ohchang-kwon
- priyawadhwa
- tinkerborg
- tsufeki
- xanonid
- yw-liu

# v0.24.0 Release 2020-07-01
This is the 24th release of Kaniko!

## New Features
* Add a new run command along with a new flag [#1300](https://github.com/GoogleContainerTools/kaniko/pull/1300)
* Add redo snapshotter.  [#1301](https://github.com/GoogleContainerTools/kaniko/pull/1301)
* Add pkg.dev to automagic config file population [#1328](https://github.com/GoogleContainerTools/kaniko/pull/1328)
* kaniko now clone git repositories recursing submodules by default [#1320](https://github.com/GoogleContainerTools/kaniko/pull/1320)

## Bug Fixes
* Fix README.md [#1323](https://github.com/GoogleContainerTools/kaniko/pull/1323)
* Fix docker-credential-gcr owner and group id [#1307](https://github.com/GoogleContainerTools/kaniko/pull/1307)

## Refactors
* check file changed in loop [#1302](https://github.com/GoogleContainerTools/kaniko/pull/1302)
* ADD GCB benchmark code [#1299](https://github.com/GoogleContainerTools/kaniko/pull/1299)
* benchmark FileSystem snapshot project added [#1288](https://github.com/GoogleContainerTools/kaniko/pull/1288)
* [Perf] Reduce loops over files when taking FS snapshot. [#1283](https://github.com/GoogleContainerTools/kaniko/pull/1283)
* Fix README.md [#1323](https://github.com/GoogleContainerTools/kaniko/pull/1323)
* Fix docker-credential-gcr owner and group id [#1307](https://github.com/GoogleContainerTools/kaniko/pull/1307)
* benchmark FileSystem snapshot project added [#1288](https://github.com/GoogleContainerTools/kaniko/pull/1288)
* [Perf] Reduce loops over files when taking FS snapshot. [#1283](https://github.com/GoogleContainerTools/kaniko/pull/1283)

Huge thank you for this release towards our contributors:
- Alexander Sharov
- Alex Szakaly
- Anthony Davies
- Art Begolli
- Batuhan Apaydın
- Ben Einaudi
- Carlos Alexandro Becker
- Carlos Sanchez
- Chris Sng
- Cole Wippern
- cvgw
- Daniel Marks
- Dani Raznikov
- David Dooling
- DracoBlue
- Gábor Lipták
- Gabriel Virga
- Gilbert Gilb's
- Giovan Isa Musthofa
- James Ravn
- Jon Henrik Bjørnstad
- Jon Johnson
- Jordan Goasdoué
- Liubov Grinkevich
- Logan.Price
- Lukasz Jakimczuk
- Mehdi Abaakouk
- Michel Hollands
- Mitchell Friedman
- Moritz Wanzenböck
- ohchang-kwon
- Or Sela
- PhoenixMage
- priyawadhwa
- Sam Stoelinga
- Tejal Desai
- Thomas Bonfort
- Thomas Stromberg
- Thomas Strömberg
- tinkerborg
- Tom Prince
- Vincent Latombe
- Wietse Muizelaar
- xanonid
- Yoan Blanc
- Yoriyasu Yano
- Yuheng Zhang
- yw-liu


# v0.23.0 Release 2020-06-04
This is the 23rd release of Kaniko!

## Bug Fixes
* Resolving nested meta ARGs [#1260](https://github.com/GoogleContainerTools/kaniko/pull/1260)
* add 64 busybox [#1254](https://github.com/GoogleContainerTools/kaniko/pull/1254)
* Apply dockefile exclude only for first stage [#1234](https://github.com/GoogleContainerTools/kaniko/pull/1234)

## New Features
* Add /etc/nsswitch.conf for /etc/hosts name resolution [#1251](https://github.com/GoogleContainerTools/kaniko/pull/1251)
* Add ability to set git auth token using environment variables [#1263](https://github.com/GoogleContainerTools/kaniko/pull/1263)
* Add retries to image push. [#1258](https://github.com/GoogleContainerTools/kaniko/pull/1258)
* Update docker-credential-gcr to support auth with GCP Artifact Registry [#1255](https://github.com/GoogleContainerTools/kaniko/pull/1255)

## Updates and Refactors
* Added integration test for multi level argument [#1285](https://github.com/GoogleContainerTools/kaniko/pull/1285)
* rename whitelist to ignorelist [#1295](https://github.com/GoogleContainerTools/kaniko/pull/1295)
* Remove direct use of DefaultTransport [#1221](https://github.com/GoogleContainerTools/kaniko/pull/1221)
* fix switching to non existent workdir [#1253](https://github.com/GoogleContainerTools/kaniko/pull/1253)
* remove duplicates save for the same dir [#1252](https://github.com/GoogleContainerTools/kaniko/pull/1252)
* add timings for resolving paths [#1284](https://github.com/GoogleContainerTools/kaniko/pull/1284)

## Documentation
* Instructions for using stdin with kubectl [#1289](https://github.com/GoogleContainerTools/kaniko/pull/1289)
* Add GoReportCard badge to README [#1249](https://github.com/GoogleContainerTools/kaniko/pull/1249)
* Make support clause more bold. [#1273](https://github.com/GoogleContainerTools/kaniko/pull/1273)
* Correct typo [#1250](https://github.com/GoogleContainerTools/kaniko/pull/1250)
* docs: add registry-certificate flag to readme [#1276](https://github.com/GoogleContainerTools/kaniko/pull/1276)

Huge thank you for this release towards our contributors:
- Anthony Davies
- Art Begolli
- Batuhan Apaydın
- Ben Einaudi
- Carlos Alexandro Becker
- Carlos Sanchez
- Chris Sng
- Cole Wippern
- Dani Raznikov
- Daniel Marks
- David Dooling
- DracoBlue
- Gabriel Virga
- Gilbert Gilb's
- Giovan Isa Musthofa
- Gábor Lipták
- James Ravn
- Jon Henrik Bjørnstad
- Jordan GOASDOUE
- Liubov Grinkevich
- Logan.Price
- Lukasz Jakimczuk
- Mehdi Abaakouk
- Michel Hollands
- Mitchell Friedman
- Moritz Wanzenböck
- Or Sela
- PhoenixMage
- Sam Stoelinga
- Tejal Desai
- Thomas Bonfort
- Thomas Stromberg
- Thomas Strömberg
- Tom Prince
- Vincent Latombe
- Wietse Muizelaar
- Yoan Blanc
- Yoriyasu Yano
- Yuheng Zhang
- cvgw
- ohchang-kwon
- tinkerborg
- xanonid
- yw-liu

# v0.22.0 Release 2020-05-07
This is a minor release of kaniko fixing:
- GCB Authentication issue
  [1242](https://github.com/GoogleContainerTools/kaniko/issues/1242)
- Re-added files if removed from base imaged [1236](https://github.com/GoogleContainerTools/kaniko/issues/1236)

Big thanks to
- David Dooling

# v0.21.0 Release - 2020-05-04
This is the 21th release of Kaniko! Thank you for patience.

This is minor release which fixes the `/kaniko/.docker` being removed in executor image
* Fixes #1227 - Readded the `/kaniko/.docker` directory [#1230](https://github.com/GoogleContainerTools/kaniko/pull/1230)

# v0.20.0 Release - 2020-05-04
This is the 20th release of Kaniko! Thank you for patience.
Please give us feedback on how we are doing by taking a short [5 question survey](https://forms.gle/HhZGEM33x4FUz9Qa6)

In this release, the highlights are:
1. Fix doubling cache layers size and error due to duplicate files in cached layers
1. Kaniko now supports reading a tar context from a stdin using `--context=tar:/.
1. Kaniko adds a new flag `--context-sub-path` to represent a subpath within the given context
1. Skip buiklding unused stages using `--skip-unused-stages` flags.

## Bug Fixes
* Snapshot FS on first cache miss. [#1214](https://github.com/GoogleContainerTools/kaniko/pull/1214)
* Add secondary group impersonation w/ !cgo support  [#1164](https://github.com/GoogleContainerTools/kaniko/pull/1164)
* kaniko generates images that docker supports in the presence of dangling symlinks [#1193](https://github.com/GoogleContainerTools/kaniko/pull/1193)
* Handle `MAINTAINERS` when passing `--single-snapshot`. [#1192](https://github.com/GoogleContainerTools/kaniko/pull/1192)
* Multistage ONBUILD COPY Support [#1190](https://github.com/GoogleContainerTools/kaniko/pull/1190)
* fix previous name checking in 'executor.build.fetchExtraStages' [#1167](https://github.com/GoogleContainerTools/kaniko/pull/1167)
* Always add parent directories of files to snapshots. [#1166](https://github.com/GoogleContainerTools/kaniko/pull/1166)
* Fix `workdir` command pointing to relative dir in first command.
* fix stages are now resolved correctly when `--skip-unused-stages` is used

## New Features
* Add ability to use public GCR repos without being authenticated [#1140](https://github.com/GoogleContainerTools/kaniko/pull/1140)
* Add timestamp to logs [#1211](https://github.com/GoogleContainerTools/kaniko/pull/1211)
* Add http support for git repository context [#1196](https://github.com/GoogleContainerTools/kaniko/pull/1196)
* Kaniko now resolves args from all stages [#1160](https://github.com/GoogleContainerTools/kaniko/pull/1160)
* kaniko adds a new flag `--context-sub-path` to represent a subpath within the given context
* feat: allow injecting through stdin tar.gz on kaniko [#1139](https://github.com/GoogleContainerTools/kaniko/pull/1139)
* Set image platform for any build [#1130](https://github.com/GoogleContainerTools/kaniko/pull/1130)
* Add --log-format parameter to README.md [#1216](https://github.com/GoogleContainerTools/kaniko/pull/1216)
* feat: multistages now respect dependencies without building unnecessary stages [#1165](https://github.com/GoogleContainerTools/kaniko/pull/1165)

## Refactors and Updates
* Refactor Kaniko to test across multistages [#1155](https://github.com/GoogleContainerTools/kaniko/pull/1155)
* upgrade go container registry to latest master [#1146](https://github.com/GoogleContainerTools/kaniko/pull/1146)
* small perf optimizing. Only remove whiteout path if it needs to be included in base image [#1147](https://github.com/GoogleContainerTools/kaniko/pull/1147)
* Don't generate cache key, if not caching builds. [#1194](https://github.com/GoogleContainerTools/kaniko/pull/1194)
* Set very large logs to Trace level [#1203](https://github.com/GoogleContainerTools/kaniko/pull/1203)
* optimize: don't parse Dockerfile twice, reusing stages [#1174](https://github.com/GoogleContainerTools/kaniko/pull/1174)
* 32bit overflow fix [#1168](https://github.com/GoogleContainerTools/kaniko/pull/1168)

## Documentation
* Update Pushing to Docker Hub to use v2 api [#1204](https://github.com/GoogleContainerTools/kaniko/pull/1204)
* Fix line endings in shell script [#1199](https://github.com/GoogleContainerTools/kaniko/pull/1199)

Huge thank you for this release towards our contributors:
- Anthony Davies
- Batuhan Apaydın
- Ben Einaudi
- Carlos Alexandro Becker
- Carlos Sanchez
- Cole Wippern
- cvgw
- Dani Raznikov
- DracoBlue
- Gilbert Gilb's
- Giovan Isa Musthofa
- James Ravn
- Jon Henrik Bjørnstad
- Jordan GOASDOUE
- Jordan Goasdoué
- Liubov Grinkevich
- Logan.Price
- Michel Hollands
- Moritz Wanzenböck
- ohchang-kwon
- Or Sela
- PhoenixMage
- Sam Stoelinga
- Tejal Desai
- Thomas Bonfort
- Thomas Stromberg
- Thomas Strömberg
- tinkerborg
- Tom Prince
- Vincent Latombe
- Wietse Muizelaar
- xanonid
- Yoan Blanc
- Yuheng Zhang
- yw-liu


# v0.19.0 Release - 2020-03-18
This is the 19th release of Kaniko!

In this release, the highlights are:
1. Cache layer size duplication regression in v0.18.0 is fixed. [#1138](https://github.com/GoogleContainerTools/kaniko/issues/1138)
1. Cache performance when using build-args. `build-args` are only part of cache key for a layer if it is used.
1. Kaniko can support a `tar.gz` context with `tar://` prefix.
1. Users can provide registry certificates for private registries.

## Bug Fixes
* Use the correct name for acr helper [#1121](https://github.com/GoogleContainerTools/kaniko/pull/1121)
* remove build args from composite key and replace all build args [#1085](https://github.com/GoogleContainerTools/kaniko/pull/1085)
* fix resolve link for dirs with trailing / [#1113](https://github.com/GoogleContainerTools/kaniko/pull/1113)

## New Features
* feat: add support of local '.tar.gz' file inside the kaniko container [#1115](https://github.com/GoogleContainerTools/kaniko/pull/1115)
* Add support to `--chown` flag to ADD command (Issue #57) [#1134](https://github.com/GoogleContainerTools/kaniko/pull/1134)
* executor: add --label flag [#1075](https://github.com/GoogleContainerTools/kaniko/pull/1075)
* Allow user to provide registry certificate [#1037](https://github.com/GoogleContainerTools/kaniko/pull/1037)

## Refactors And Updates
* Migrate to golang 1.14 [#1098](https://github.com/GoogleContainerTools/kaniko/pull/1098)
* Make cloudbuild.yaml re-usable for anyone [#1135](https://github.com/GoogleContainerTools/kaniko/pull/1135)
* fix: credential typo [#1128](https://github.com/GoogleContainerTools/kaniko/pull/1128)
* Travis k8s integration test [#1124](https://github.com/GoogleContainerTools/kaniko/pull/1124)
* Add more tests for Copy and some fixes. [#1114](https://github.com/GoogleContainerTools/kaniko/pull/1114)

## Documentation
* Update README on running in Docker [#1141](https://github.com/GoogleContainerTools/kaniko/pull/1141)

Huge thank you for this release towards our contributors:
 - Anthony Davies
 - Batuhan Apaydın
 - Ben Einaudi
 - Carlos Sanchez
 - Cole Wippern
 - cvgw
 - Dani Raznikov
 - DracoBlue
 - James Ravn
 - Jordan GOASDOUE
 - Logan.Price
 - Moritz Wanzenböck
 - ohchang-kwon
 - Or Sela
 - Sam Stoelinga
 - Tejal Desai
 - Thomas Bonfort
 - Thomas Strömberg
 - tinkerborg
 - Wietse Muizelaar
 - xanonid
 - Yoan Blanc
 - Yuheng Zhang

 # v0.18.0 Release -2020-03-05
This release fixes all the regression bugs associated with v0.17.0 and v0.17.1.
This release, the team did a lot of work improving our test infrastructure, more tests cases
and refactored filesystem walking.

Thank you all for your patience and supporting us throughout!

## Bug Fixes
* fix home being reset to root [#1072](https://github.com/GoogleContainerTools/kaniko/pull/1072)
* fix user metadata set to USER:GROUP if group string is not set [#1105](https://github.com/GoogleContainerTools/kaniko/pull/1105)
* check for filepath.Walk error everywhere [#1086](https://github.com/GoogleContainerTools/kaniko/pull/1086)
* fix #1092 TestRelativePaths [#1093](https://github.com/GoogleContainerTools/kaniko/pull/1093)
* Resolve filepaths before scanning for changes [#1069](https://github.com/GoogleContainerTools/kaniko/pull/1069)
* Fix #1020 os.Chtimes invalid arg [#1074](https://github.com/GoogleContainerTools/kaniko/pull/1074)
* Fix #1067 - image no longer available [#1068](https://github.com/GoogleContainerTools/kaniko/pull/1068)
* Ensure image SHA stays consistent when layer contents haven't changed [#1032](https://github.com/GoogleContainerTools/kaniko/pull/1032)
* fix flake TestRun/Dockerfile_test_copy_symlink [#1030](https://github.com/GoogleContainerTools/kaniko/pull/1030)

## New Features
* root: add --registry-mirror flag [#836](https://github.com/GoogleContainerTools/kaniko/pull/836)
* set log format using a flag [#1031](https://github.com/GoogleContainerTools/kaniko/pull/1031)
* Do not recompute layers retrieved from cache [#882](https://github.com/GoogleContainerTools/kaniko/pull/882)
* More idiomatic logging config [#1040](https://github.com/GoogleContainerTools/kaniko/pull/1040)


## Test Refactors and Updates
* Split travis integration tests [#1090](https://github.com/GoogleContainerTools/kaniko/pull/1090)
* Add integration tests from Issues [#1054](https://github.com/GoogleContainerTools/kaniko/pull/1054)
* add integration tests with their own context [#1088](https://github.com/GoogleContainerTools/kaniko/pull/1088)
* Fixed typo in README.md [#1060](https://github.com/GoogleContainerTools/kaniko/pull/1060)
* test: refactor container-diff call [#1077](https://github.com/GoogleContainerTools/kaniko/pull/1077)
* Refactor integration image built [#1049](https://github.com/GoogleContainerTools/kaniko/pull/1049)
* separate travis into multiple jobs for parallelization [#1055](https://github.com/GoogleContainerTools/kaniko/pull/1055)
* refactor copy.chown code and add more tests [#1027](https://github.com/GoogleContainerTools/kaniko/pull/1027)
* Allow contributors to launch integration tests against local registry [#1014](https://github.com/GoogleContainerTools/kaniko/pull/1014)

## Documentation
* add design proposal template [#1046](https://github.com/GoogleContainerTools/kaniko/pull/1046)
* Update filesystem proposal status to Reviewed [#1066](https://github.com/GoogleContainerTools/kaniko/pull/1066)
* update instructions for running integration tests [#1034](https://github.com/GoogleContainerTools/kaniko/pull/1034)
* design proposal 01: filesystem resolution [#1048](https://github.com/GoogleContainerTools/kaniko/pull/1048)
* Document that this tool is not officially supported by Google [#1044](https://github.com/GoogleContainerTools/kaniko/pull/1044)
* Fix example pod.yml to not mount to root [#1043](https://github.com/GoogleContainerTools/kaniko/pull/1043)
* fixing docker run command in README.md [#1103](https://github.com/GoogleContainerTools/kaniko/pull/1103)

Huge thank you for this release towards our contributors:
- Anthony Davies
- Batuhan Apaydın
- Ben Einaudi
- Cole Wippern
- cvgw
- DracoBlue
- James Ravn
- Logan.Price
- Moritz Wanzenböck
- ohchang-kwon
- Or Sela
- Sam Stoelinga
- Tejal Desai
- Thomas Bonfort
- Thomas Strömberg
- tinkerborg
- Wietse Muizelaar
- xanonid
- Yoan Blanc

# v0.17.1 Release - 2020-02-04

This is minor patch release to fix [#1002](https://github.com/GoogleContainerTools/kaniko/issues/1002)

# v0.17.0 Release - 2020-02-03

## New Features
* Expand build argument from environment when no value specified [#993](https://github.com/GoogleContainerTools/kaniko/pull/993)
* whitelist  /tmp/apt-key-gpghome.* directory [#1000](https://github.com/GoogleContainerTools/kaniko/pull/1000)
* Add flag to `--whitelist-var-run` set to true to preserver default kani… [#1011](https://github.com/GoogleContainerTools/kaniko/pull/1011)
* Prefer platform that is currently running for pulling remote images and kaniko binary Makefile target [#980](https://github.com/GoogleContainerTools/kaniko/pull/980)

## Bug Fixes
* Fix caching to respect .dockerignore [#854](https://github.com/GoogleContainerTools/kaniko/pull/854)
* Fixes #988 run_in_docker.sh only works with gcr.io [#990](https://github.com/GoogleContainerTools/kaniko/pull/990)
* Fix Symlinks not being copied across stages [#971](https://github.com/GoogleContainerTools/kaniko/pull/971)
* Fix home and group set for user command [#995](https://github.com/GoogleContainerTools/kaniko/pull/995)
* Fix COPY or ADD to symlink destination breaks image [#943](https://github.com/GoogleContainerTools/kaniko/pull/943)
* [Caching] Fix bug with deleted files and cached run and copy commands
* [Mutistage Build] Fix bug with capital letter in stage names [#983](https://github.com/GoogleContainerTools/kaniko/pull/983)
* Fix #940 set modtime when extracting [#981](https://github.com/GoogleContainerTools/kaniko/pull/981)
* Fix Ability for ADD to unTar a file [#792](https://github.com/GoogleContainerTools/kaniko/pull/792)

## Updates and Refactors
* fix test flake [#1016](https://github.com/GoogleContainerTools/kaniko/pull/1016)
* Upgrade go-containerregistry third-party library [#957](https://github.com/GoogleContainerTools/kaniko/pull/957)
* Remove debug tag being built for every push to master [#1004](https://github.com/GoogleContainerTools/kaniko/pull/1004)
* Run integration tests in Travis CI [#979](https://github.com/GoogleContainerTools/kaniko/pull/979)


Huge thank you for this release towards our contributors:
- Anthony Davies
- Ben Einaudi
- Cole Wippern
- cvgw
- Logan.Price
- Moritz Wanzenböck
- ohchang-kwon
- Sam Stoelinga
- Tejal Desai
- Thomas Bonfort
- Wietse Muizelaar

# v0.16.0 Release - 2020-01-17

Happy New Year 2020!

## Bug Fixes
* Support for private registries in the cache warmer [#941](https://github.com/GoogleContainerTools/kaniko/pull/941)
* Fix bug with docker compatibility ArgsEscaped [#964](https://github.com/GoogleContainerTools/kaniko/pull/964)
* Clean code (Condition is always 'false' because 'err' is always 'nil' ). [#967](https://github.com/GoogleContainerTools/kaniko/pull/967)
* Fix #647 Copy dir permissions [#961](https://github.com/GoogleContainerTools/kaniko/pull/961)
* Allow setting serviceAccount in integration test [#965](https://github.com/GoogleContainerTools/kaniko/pull/965)
* Fix #926 cache warmer and method signature [#927](https://github.com/GoogleContainerTools/kaniko/pull/927)
* Fix #948 update valid license years [#949](https://github.com/GoogleContainerTools/kaniko/pull/949)
* Move hash bang to first line. [#954](https://github.com/GoogleContainerTools/kaniko/pull/954)
* Fix #944 include docker-credential-acr-linux [#945](https://github.com/GoogleContainerTools/kaniko/pull/945)
* Fix #925 broken insecure pull [#932](https://github.com/GoogleContainerTools/kaniko/pull/932)
* Push to ECR using instance roles [#930](https://github.com/GoogleContainerTools/kaniko/pull/930)
* Upgrade aws go sdk for supporting eks oidc credential chain [#832](https://github.com/GoogleContainerTools/kaniko/pull/832)
* Push image [#866](https://github.com/GoogleContainerTools/kaniko/pull/866)

## Updates and Refactors
* Fixes #950 integration test failing on go 1.13 [#955](https://github.com/GoogleContainerTools/kaniko/pull/955)
* Tidy dependencies [#939](https://github.com/GoogleContainerTools/kaniko/pull/939)
* changing to modules from dependencies [#869](https://github.com/GoogleContainerTools/kaniko/pull/869)
* Changing Log to trace [#920](https://github.com/GoogleContainerTools/kaniko/pull/920)

## Documentation
* docs: fix document on DoBuild [#668](https://github.com/GoogleContainerTools/kaniko/pull/668)
* Update outdated toc in README.md [#867](https://github.com/GoogleContainerTools/kaniko/pull/867)

Huge thank you for this release towards our contributors:
- Adrian Mouat
- Balint Pato
- Ben Einaudi
- Benjamin EINAUDI
- Carlos Sanchez
- Cole Wippern
- Daniel Strobusch
- Eduard Laur
- Fahri Yardımcı
- Josh Soref
- lou-lan
- Nao YONASHIRO
- poy
- Prashant Arya
- priyawadhwa
- Pweetoo
- Remko van Hunen
- Sam Stoelinga
- Stijn De Haes
- Tejal Desai
- tommaso.doninelli
- Will Ripley


# v0.15.0 Release - 2019-12-20

## Bug fixes
* Fix #899 cached copy results in inconsistent key [#914](https://github.com/GoogleContainerTools/kaniko/pull/914)
* Fix contribution issue sentence [#912](https://github.com/GoogleContainerTools/kaniko/pull/912)
* Include source stage cache key in cache key for COPY commands using --from [#883](https://github.com/GoogleContainerTools/kaniko/pull/883)
* Fix failure when using capital letters in image alias in 'FROM ... AS…' instruction [#839](https://github.com/GoogleContainerTools/kaniko/pull/839)
* Add golangci.yaml file matching current config [#893](https://github.com/GoogleContainerTools/kaniko/pull/893)
* when copying, skip files with the same name [#905](https://github.com/GoogleContainerTools/kaniko/pull/905)
* Modified error message for writing image with digest file [#849](https://github.com/GoogleContainerTools/kaniko/pull/849)
* Don't exit optimize early; record last cachekey [#892](https://github.com/GoogleContainerTools/kaniko/pull/892)
* Final cachekey for stage [#891](https://github.com/GoogleContainerTools/kaniko/pull/891)
* Update error handling and logging for cache [#879](https://github.com/GoogleContainerTools/kaniko/pull/879)
* Resolve symlink targets to abs path before copying [#857](https://github.com/GoogleContainerTools/kaniko/pull/857)
* Fix quote strip behavior for ARG values [#850](https://github.com/GoogleContainerTools/kaniko/pull/850)

## Updates and Refactors
* add unit tests for caching run and copy [#888](https://github.com/GoogleContainerTools/kaniko/pull/888)
* Only build required docker images for integration tests [#898](https://github.com/GoogleContainerTools/kaniko/pull/898)
* Add integration test for add url with arg [#863](https://github.com/GoogleContainerTools/kaniko/pull/863)
* Add unit tests for compositecache and stagebuilder [#890](https://github.com/GoogleContainerTools/kaniko/pull/890)

## Documentation
* updated readme [#906](https://github.com/GoogleContainerTools/kaniko/pull/906)
* nits in README [#861](https://github.com/GoogleContainerTools/kaniko/pull/861)
* Invalid link to missing file config.json [#876](https://github.com/GoogleContainerTools/kaniko/pull/876)
* Fix README.md anchor links [#872](https://github.com/GoogleContainerTools/kaniko/pull/872)
* Update readme known issues [#874](https://github.com/GoogleContainerTools/kaniko/pull/874)

Huge thank you for this release towards our contributors:
- Balint Pato
- Ben Einaudi
- Cole Wippern
- Eduard Laur
- Josh Soref
- Pweetoo
- Tejal Desai
- Will Ripley
- poy
- priyawadhwa
- tommaso.doninelli


# v0.14.0 Release - 2019-11-08

## New Features
* Added --image-name-with-digest flag [#841](https://github.com/GoogleContainerTools/kaniko/pull/841)
* Add support to download context file from Azure Blob Storage [#816](https://github.com/GoogleContainerTools/kaniko/pull/816)
* Add BUILD_ARGs to ease use of proxy [#810](https://github.com/GoogleContainerTools/kaniko/pull/810)

## Bug Fixes
* fix tests for default home [#824](https://github.com/GoogleContainerTools/kaniko/pull/824)
* Issue #439 Strip out double quotes in ARG value [#834](https://github.com/GoogleContainerTools/kaniko/pull/834)
* Fixes caching with COPY command [#773](https://github.com/GoogleContainerTools/kaniko/pull/773)
* 828: clean up docker doc, fix context var in run cmd [#829](https://github.com/GoogleContainerTools/kaniko/pull/829)
* fix build_args in MakeFile, have Travis run make images to preven issue in future [#821](https://github.com/GoogleContainerTools/kaniko/pull/821)

## Updates and Refactors
* changing debug to trace [#825](https://github.com/GoogleContainerTools/kaniko/pull/825)

## Documentation
* Details about --tarPath usage improved [#811](https://github.com/GoogleContainerTools/kaniko/pull/811)


# v0.13.0 Release - 2019-10-04

## New Features
* Add `kaniko version` command [#796](https://github.com/GoogleContainerTools/kaniko/pull/796)
* Write data about pushed images for GCB kaniko build step if env var `BUILDER_OUTPUT` is set [#602](https://github.com/GoogleContainerTools/kaniko/pull/602)
* Support `Dockerfile.dockerignore` relative to `Dockerfile` [#801](https://github.com/GoogleContainerTools/kaniko/pull/801)

## Bug Fixes
* fix creating abs path for urls [#804](https://github.com/GoogleContainerTools/kaniko/pull/804)
* Fix #691 - ADD does not understand ENV variables [#768](https://github.com/GoogleContainerTools/kaniko/pull/768)
* Resolve relative paths to absolute paths in command line arguments [#736](https://github.com/GoogleContainerTools/kaniko/pull/736)
* insecure flag is now honored with `--cache` flag. [#685](https://github.com/GoogleContainerTools/kaniko/pull/685)
* Reduce log level for adding file message [#624](https://github.com/GoogleContainerTools/kaniko/pull/624)
* Fix SIGSEGV on file system deletion while building [#765](https://github.com/GoogleContainerTools/kaniko/pull/765)

## Updates and Refactors
* add debug level info what is the layer type [#805](https://github.com/GoogleContainerTools/kaniko/pull/805)
* Update base image to golang:1.12 [#648](https://github.com/GoogleContainerTools/kaniko/pull/648)
* Add some triage notes to issue template. [#794](https://github.com/GoogleContainerTools/kaniko/pull/794)
* double help text about skip-verify-tls [#782](https://github.com/GoogleContainerTools/kaniko/pull/782)
* Add a pull request template [#795](https://github.com/GoogleContainerTools/kaniko/pull/795)
* Correct CheckPushPermission comment. [#671](https://github.com/GoogleContainerTools/kaniko/pull/671)

## Documentation
* Use kaniko with docker config.json password [#129](https://github.com/GoogleContainerTools/kaniko/pull/129)
* Add getting started tutorial [#790](https://github.com/GoogleContainerTools/kaniko/pull/790)

## Performance
* feat: optimize build [#694](https://github.com/GoogleContainerTools/kaniko/pull/694)

Huge thank you for this release towards our contributors:
- alexa
- Andreas Bergmeier
- Carlos Alexandro Becker
- Carlos Sanchez
- chhsia0
- debuggy
- Deniz Zoeteman
- Don McCasland
- Fred Cox
- Herrmann Hinz
- Hugues Alary
- Jason Hall
- Johannes 'fish' Ziemke
- jonjohnsonjr
- Luke Wood
- Matthew Dawson
- Mingliang Tao
- Monard Vong
- Nao YONASHIRO
- Niels Denissen
- Prashant
- priyawadhwa
- Priya Wadhwa
- Sascha Askani
- sharifelgamal
- Sharif Elgamal
- Takeaki Matsumoto
- Taylor Barrella
- Tejal Desai
- Thao-Nguyen Do
- tralexa
- Victor Noel
- v.rul
- Warren Seymour
- xanonid
- Xueshan Feng
- Антон Костенко
- Роман Небалуев

# v0.12.0 Release - 2019-09/13

## New Features
* Added `--oci-layout-path` flag to save image in OCI layout. [#744](https://github.com/GoogleContainerTools/kaniko/pull/744)
* Add support for S3 custom endpoint [#698](https://github.com/GoogleContainerTools/kaniko/pull/698)

## Bug Fixes
* Setting PATH [#760](https://github.com/GoogleContainerTools/kaniko/pull/760)
* Remove leading slash in layer tarball paths (Closes: #726) [#729](https://github.com/GoogleContainerTools/kaniko/pull/729)

## Updates and Refactors
* Remove cruft [#635](https://github.com/GoogleContainerTools/kaniko/pull/635)
* Add desc for `--skip-tls-verify-pull` to README [#493](https://github.com/GoogleContainerTools/kaniko/pull/493)

Huge thank you for this release towards our contributors:
- Carlos Alexandro Becker
- Carlos Sanchez
- chhsia0
- Deniz Zoeteman
- Luke Wood
- Matthew Dawson
- Niels Denissen
- Priya Wadhwa
- Sharif Elgamal
- Takeaki Matsumoto
- Taylor Barrella
- Tejal Desai
- v.rul
- Warren Seymour
- xanonid
- Xueshan Feng
- Роман Небалуев


# v0.11.0 Release - 2019-08-23

## Bug Fixes
* fix unpacking archives via ADD [#717](https://github.com/GoogleContainerTools/kaniko/pull/717)
* Reverted not including build args in cache key [#739](https://github.com/GoogleContainerTools/kaniko/pull/739)
* Create cache directory if it doesn't already exist [#452](https://github.com/GoogleContainerTools/kaniko/pull/452)

## New Features
* add multiple user agents to kaniko if upstream_client_type value  is set [#750](https://github.com/GoogleContainerTools/kaniko/pull/750)
* Make container layers captured using FS snapshots reproducible [#714](https://github.com/GoogleContainerTools/kaniko/pull/714)
* Include warmer in debug image [#497](https://github.com/GoogleContainerTools/kaniko/pull/497)
* Bailout when there is not enough input arguments [#735](https://github.com/GoogleContainerTools/kaniko/pull/735)
* Add checking image presence in cache prior to downloading it [#723](https://github.com/GoogleContainerTools/kaniko/pull/723)

## Additonal PRs
* Document how to build from git reference [#730](https://github.com/GoogleContainerTools/kaniko/pull/730)
* Misc. small changes/refactoring [#712](https://github.com/GoogleContainerTools/kaniko/pull/712)
* Update go-containerregistry [#680](https://github.com/GoogleContainerTools/kaniko/pull/680)
* Update version of go-containerregistry [#724](https://github.com/GoogleContainerTools/kaniko/pull/724)
* feat: support specifying branch for cloning [#703](https://github.com/GoogleContainerTools/kaniko/pull/703)

Huge thank you for this release towards our contributors:
- Carlos Alexandro Becker
- Carlos Sanchez
- Deniz Zoeteman
- Luke Wood
- Matthew Dawson
- priyawadhwa
- sharifelgamal
- Sharif Elgamal
- Taylor Barrella
- Tejal Desai
- v.rul
- Warren Seymour
- Xueshan Feng
- Роман Небалуе

# v0.10.0 Release - 2019-06-19

## Bug Fixes
* Fix kaniko caching [#639](https://github.com/GoogleContainerTools/kaniko/pull/639)
* chore: fix typo [#665](https://github.com/GoogleContainerTools/kaniko/pull/665)
* Fix file mode bug [#618](https://github.com/GoogleContainerTools/kaniko/pull/618)
* Fix arg handling for multi-stage images in COPY instructions. [#621](https://github.com/GoogleContainerTools/kaniko/pull/621)
* Fix parent directory permissions [#619](https://github.com/GoogleContainerTools/kaniko/pull/619)
* Environment variables should be replaced in URLs in ADD commands. [#580](https://github.com/GoogleContainerTools/kaniko/pull/580)
* Update the cache warmer to also save manifests. [#576](https://github.com/GoogleContainerTools/kaniko/pull/576)
* Fix typo in error message [#569](https://github.com/GoogleContainerTools/kaniko/pull/569)

## New Features
* Add SkipVerify support to CheckPushPermissions. [#663](https://github.com/GoogleContainerTools/kaniko/pull/663)
* Creating  github Build Context [#672](https://github.com/GoogleContainerTools/kaniko/pull/672)
* Add `--digest-file` flag to output built digest to file. [#655](https://github.com/GoogleContainerTools/kaniko/pull/655)
* README.md: update BuildKit/img comparison [#642](https://github.com/GoogleContainerTools/kaniko/pull/642)
* Add documentation for --verbosity flag [#634](https://github.com/GoogleContainerTools/kaniko/pull/634)
* Optimize file copying and stage saving between stages. [#605](https://github.com/GoogleContainerTools/kaniko/pull/605)
* Add an integration test for USER unpacking. [#600](https://github.com/GoogleContainerTools/kaniko/pull/600)
* Added missing documentation for --skip-tls-verify-pull arg [#593](https://github.com/GoogleContainerTools/kaniko/pull/593)
* README.me: update Buildah description [#586](https://github.com/GoogleContainerTools/kaniko/pull/586)
* Add missing tests for bucket util [#565](https://github.com/GoogleContainerTools/kaniko/pull/565)
* Look for manifests in the local cache next to the full images. [#570](https://github.com/GoogleContainerTools/kaniko/pull/570)
* Make the run_in_docker script support caching. [#564](https://github.com/GoogleContainerTools/kaniko/pull/564)
* Refactor snapshotting [#561](https://github.com/GoogleContainerTools/kaniko/pull/561)
* Stop storing a separate cache hash. [#560](https://github.com/GoogleContainerTools/kaniko/pull/560)
* Speed up workdir by always returning an empty filelist (rather than a… [#557](https://github.com/GoogleContainerTools/kaniko/pull/557)
* Refactor whitelist handling. [#559](https://github.com/GoogleContainerTools/kaniko/pull/559)
* Refactor the build loop to fetch stagebuilders earlier. [#558](https://github.com/GoogleContainerTools/kaniko/pull/558)

## Additonal PRs
* Improve changelog dates [#657](https://github.com/GoogleContainerTools/kaniko/pull/657)
* Change verbose output from info to debug [#640](https://github.com/GoogleContainerTools/kaniko/pull/640)
* Check push permissions before building images [#622](https://github.com/GoogleContainerTools/kaniko/pull/622)
* Bump go-containerregistry to 8c1640add99804503b4126abc718931a4d93c31a [#609](https://github.com/GoogleContainerTools/kaniko/pull/609)
* Update go-containerregistry [#599](https://github.com/GoogleContainerTools/kaniko/pull/599)
* Log "Skipping paths under..." to debug [#571](https://github.com/GoogleContainerTools/kaniko/pull/571)

Huge thank you for this release towards our contributors:
- Achilleas Pipinellis
- Adrian Duong
- Akihiro Suda
- Andreas Bergmeier
- Andrew Rynhard
- Anthony Weston
- Anurag Goel
- Balint Pato
- Christie Wilson
- Daisuke Taniwaki
- Dan Cecile
- Dirk Gustke
- dlorenc
- Fredrik Lönnegren
- Gijs
- Jake Shadle
- James Rawlings
- Jason Hall
- Johan Hernandez
- Johannes 'fish' Ziemke
- Kartik Verma
- linuxshokunin
- MMeent
- Myers Carpenter
- Nándor István Krácser
- Nao YONASHIRO
- Priya Wadhwa
- Sharif Elgamal
- Shuhei Kitagawa
- Valentin Rothberg
- Vincent Demeester

# v0.9.0 Release - 2019-02-08

## Bug Fixes
* Bug fix with volumes declared in base images during multi-stage builds
* Bug fix during snapshotting multi-stage builds.
* Bug fix for caching with tar output.

# v0.8.0 Release - 2019-01-29

## New Features
* Even faster snapshotting with godirwalk
* Added TTL for caching

## Updates
* Change cache key calculation to be more reproducible.
* Make the Digest calculation faster for locally-cached images.
* Simplify snapshotting.

## Bug Fixes
* Fix bug with USER command and unpacking base images.
* Added COPY --from=previous stage name/number validation

# v0.7.0 Release - 2018-12-10

## New Features
* Add support for COPY --from an unrelated image

## Updates
* Speed up snapshotting by using filepath.SkipDir
* Improve layer cache upload performance
* Skip unpacking the base image in certain cases

## Bug Fixes
* Fix bug with call loop
* Fix caching for multi-step builds

# v0.6.0 Release - 2018-11-06

## New Features
* parse arg commands at the top of dockerfiles [#404](https://github.com/GoogleContainerTools/kaniko/pull/404)
* Add buffering for large layers. [#428](https://github.com/GoogleContainerTools/kaniko/pull/428)
* Separate Insecure Pull Options [#409](https://github.com/GoogleContainerTools/kaniko/pull/409)
* Add support for .dockerignore file [#394](https://github.com/GoogleContainerTools/kaniko/pull/394)
* Support insecure pull [#401](https://github.com/GoogleContainerTools/kaniko/pull/401)

## Updates
* Preserve options when doing a cache push [#423](https://github.com/GoogleContainerTools/kaniko/pull/423)
* More cache cleanups: [#397](https://github.com/GoogleContainerTools/kaniko/pull/397)
*  adding documentation for base image caching [#421](https://github.com/GoogleContainerTools/kaniko/pull/421)
* Update go-containerregistry [#420](https://github.com/GoogleContainerTools/kaniko/pull/420)
* Update README [#419](https://github.com/GoogleContainerTools/kaniko/pull/419)
* Use remoteImage function when getting digest for cache [#413](https://github.com/GoogleContainerTools/kaniko/pull/413)
* adding exit 1 when there are not enough command line vars passed to `… [#415](https://github.com/GoogleContainerTools/kaniko/pull/415)
* "Container Builder" - > "Cloud Build" [#414](https://github.com/GoogleContainerTools/kaniko/pull/414)
* adding the cache warmer to the release process [#412](https://github.com/GoogleContainerTools/kaniko/pull/412)

## Bug Fixes
* Fix bugs with .dockerignore and improve integration test [#424](https://github.com/GoogleContainerTools/kaniko/pull/424)
* fix releasing the cache warmer [#418](https://github.com/GoogleContainerTools/kaniko/pull/418)


# v0.5.0 Release - 2018-10-16

## New Features
* Persistent volume caching for base images [#383](https://github.com/GoogleContainerTools/kaniko/pull/383)

## Updates
* Use only the necessary files in the cache keys. [#387](https://github.com/GoogleContainerTools/kaniko/pull/387)
* Change loglevel for copying files to debug (#303) [#393](https://github.com/GoogleContainerTools/kaniko/pull/393)
* Improve IsDestDir functionality with filesystem info [#390](https://github.com/GoogleContainerTools/kaniko/pull/390)
* Refactor the build loop. [#385](https://github.com/GoogleContainerTools/kaniko/pull/385)
* Rework cache key generation a bit. [#375](https://github.com/GoogleContainerTools/kaniko/pull/375)

## Bug Fixes
* fix mispell [#396](https://github.com/GoogleContainerTools/kaniko/pull/396)
* Update go-containerregistry dependency [#388](https://github.com/GoogleContainerTools/kaniko/pull/388)
* chore: fix broken markdown (CHANGELOG.md) [#382](https://github.com/GoogleContainerTools/kaniko/pull/382)
* Don't cut everything after an equals sign [#381](https://github.com/GoogleContainerTools/kaniko/pull/381)


# v0.4.0 Release - 2018-10-01

## New Features
* Add a benchmark package to store and monitor timings. [#367](https://github.com/GoogleContainerTools/kaniko/pull/367)
* Add layer caching to kaniko [#353](https://github.com/GoogleContainerTools/kaniko/pull/353)
* Update issue templates [#340](https://github.com/GoogleContainerTools/kaniko/pull/340)
* Separate --insecure-skip-tls-verify flag into two separate flags [#311](https://github.com/GoogleContainerTools/kaniko/pull/311)
* Updated created by time for built image [#328](https://github.com/GoogleContainerTools/kaniko/pull/328)
* Add Flag to Disable Push to Container Registry [#292](https://github.com/GoogleContainerTools/kaniko/pull/292)
* Add a new flag to cleanup the filesystem at the end [#370](https://github.com/GoogleContainerTools/kaniko/pull/370)

## Updates
* Update README to add information about layer caching [#364](https://github.com/GoogleContainerTools/kaniko/pull/364)
* Suppress usage upon Run error [#356](https://github.com/GoogleContainerTools/kaniko/pull/356)
* Refactor build into stageBuilder type [#343](https://github.com/GoogleContainerTools/kaniko/pull/343)
* Replace gometalinter with GolangCI-Lint [#349](https://github.com/GoogleContainerTools/kaniko/pull/349)
* Add Key() to LayeredMap and Snapshotter [#337](https://github.com/GoogleContainerTools/kaniko/pull/337)
* Add CacheCommand to DockerCommand interface [#336](https://github.com/GoogleContainerTools/kaniko/pull/336)
* Extract filesystem in order rather than in reverse [#326](https://github.com/GoogleContainerTools/kaniko/pull/326)
* Configure logs to show colors [#327](https://github.com/GoogleContainerTools/kaniko/pull/327)
* Enable shared config for s3 [#321](https://github.com/GoogleContainerTools/kaniko/pull/321)
* Update go-containerregistry. [#305](https://github.com/GoogleContainerTools/kaniko/pull/305)
* Tag latest in cloudbuild.yaml [#287](https://github.com/GoogleContainerTools/kaniko/pull/287)
* Set default home value [#281](https://github.com/GoogleContainerTools/kaniko/pull/281)
* Update deps [#265](https://github.com/GoogleContainerTools/kaniko/pull/265)
* Update go-containerregistry dep and remove unnecessary Options [#376](https://github.com/GoogleContainerTools/kaniko/pull/376)
* Add a bit more context to layer offset failures [#264](https://github.com/GoogleContainerTools/kaniko/pull/264)

## Bug Fixes
* Whitelist /busybox in the debug image [#369](https://github.com/GoogleContainerTools/kaniko/pull/369)
* Check --cache-repo is provided with --cache and --no-push [#374](https://github.com/GoogleContainerTools/kaniko/pull/374)
* Fixes a whitelist issue when untarring files in ADD commands. [#371](https://github.com/GoogleContainerTools/kaniko/pull/371)
* set default HOME env properly [#341](https://github.com/GoogleContainerTools/kaniko/pull/341)
* Review config for cmd/entrypoint after building a stage [#348](https://github.com/GoogleContainerTools/kaniko/pull/348)
* Enable overwriting of links (solves #351) [#360](https://github.com/GoogleContainerTools/kaniko/pull/360)
* Only return stdout when running commands for integration tests [#363](https://github.com/GoogleContainerTools/kaniko/pull/363)
* Whitelist /etc/mtab [#347](https://github.com/GoogleContainerTools/kaniko/pull/347)
* Added a KanikoStage type for each stage of a Dockerfile [#320](https://github.com/GoogleContainerTools/kaniko/pull/320)
* Make sure paths are absolute before matching files to wildcard sources [#330](https://github.com/GoogleContainerTools/kaniko/pull/330)
* Build each kaniko image separately [#324](https://github.com/GoogleContainerTools/kaniko/pull/324)
* support multiple tags when writing to a tarfile [#323](https://github.com/GoogleContainerTools/kaniko/pull/323)
* Snapshot only specific files for COPY [#319](https://github.com/GoogleContainerTools/kaniko/pull/319)
* Remove some constraints from our Gopkg.toml. [#318](https://github.com/GoogleContainerTools/kaniko/pull/318)
* Always snapshot files in COPY and RUN commands [#289](https://github.com/GoogleContainerTools/kaniko/pull/289)
* Refactor command line arguments and the executor [#306](https://github.com/GoogleContainerTools/kaniko/pull/306)
* Fix bug in SaveStage function for multistage builds [#295](https://github.com/GoogleContainerTools/kaniko/pull/295)
* Get absolute path of file before checking whitelist [#293](https://github.com/GoogleContainerTools/kaniko/pull/293)
* Fix support for insecure registry [#169](https://github.com/GoogleContainerTools/kaniko/pull/169)
* ignore sockets when adding to tar [#288](https://github.com/GoogleContainerTools/kaniko/pull/288)
* fix add command bug when adding remote URLs [#277](https://github.com/GoogleContainerTools/kaniko/pull/277)
* Environment variables with multiple '=' are not parsed correctly [#278](https://github.com/GoogleContainerTools/kaniko/pull/278)
* Ensure cmd.SysProcAttr is set before modifying it [#275](https://github.com/GoogleContainerTools/kaniko/pull/275)
* Don't copy same files twice in copy integration tests [#273](https://github.com/GoogleContainerTools/kaniko/pull/273)
* Extract intermediate stages to filesystem [#266](https://github.com/GoogleContainerTools/kaniko/pull/266)
* Fix process group handling. [#271](https://github.com/GoogleContainerTools/kaniko/pull/271)
* Only add whiteout files once [#270](https://github.com/GoogleContainerTools/kaniko/pull/270)
* Fix handling of the volume directive [#334](https://github.com/GoogleContainerTools/kaniko/pull/334)


# v0.3.0 Release - 2018-07-31
## New Features
* Local integration testing [#256](https://github.com/GoogleContainerTools/kaniko/pull/256)
* Add --target flag for multistage builds [#255](https://github.com/GoogleContainerTools/kaniko/pull/255)
* Look for on cluster credentials using k8s chain [#243](https://github.com/GoogleContainerTools/kaniko/pull/243)

## Bug Fixes
* Kill grandchildren spun up by child processes [#247](https://github.com/GoogleContainerTools/kaniko/issues/247)
* Fix bug in copy command [#221](https://github.com/GoogleContainerTools/kaniko/issues/221)
* Multi-stage errors when referencing earlier stages [#233](https://github.com/GoogleContainerTools/kaniko/issues/233)


# v0.2.0 Release - 2018-07-09

## New Features
* Support for adding different source contexts, including Amazon S3 [#195](https://github.com/GoogleContainerTools/kaniko/issues/195)
* Added --reproducible [#205](https://github.com/GoogleContainerTools/kaniko/pull/205) and --single-snapshot [#204](https://github.com/GoogleContainerTools/kaniko/pull/204) flags
* Documented running kaniko in gVisor [#194](https://github.com/GoogleContainerTools/kaniko/pull/194)
* Update go-containerregistry so kaniko works better with Harbor and Gitlab[#227](https://github.com/GoogleContainerTools/kaniko/pull/227)
* Push image to multiple destinations [#184](https://github.com/GoogleContainerTools/kaniko/pull/184)

# v0.1.0 Release - 2018-05-17

## New Features
* The majority of Dockerfile commands are feature complete [#1](https://github.com/GoogleContainerTools/kaniko/issues/1)
* Support for multi-stage Dockerfile builds [#141](https://github.com/GoogleContainerTools/kaniko/pull/141)
* Refactored integration tests [#126](https://github.com/GoogleContainerTools/kaniko/pull/126)
* Added debug image with a busybox shell [#171](https://github.com/GoogleContainerTools/kaniko/pull/1710)
* Added credential helper for Amazon ECR [#167](https://github.com/GoogleContainerTools/kaniko/pull/167)

