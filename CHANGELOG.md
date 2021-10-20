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
gcr.io/kaniko-project/executor:debug-v1.5.2 and
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:slim-v1.5.2
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
gcr.io/kaniko-project/executor:debug-v1.5.1 and
```

The slim executor images which don't contain any authentication binaries are available at:
```
gcr.io/kaniko-project/executor:slim
gcr.io/kaniko-project/executor:slim-v1.5.1
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
gcr.io/kaniko-project/executor:debug-v1.5.0 and
```

In this release, we have 2 slim executor images which don't contain any authentication binaries. 

1. `gcr.io/kaniko-project/executor:slim`  &
2. `gcr.io/kaniko-project/executor:slim-v1.5.0`


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
 
