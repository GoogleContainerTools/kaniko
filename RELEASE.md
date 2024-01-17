# Kaniko Release Process

This document explains the Kaniko release process.

Kaniko is not an officially supported Google product. Kaniko is maintained by Google.


## Self-Serve Kaniko Release  [Non-Google contributors]
Kaniko is not an officially supported Google product however only contributors part of
Google organization can release Kaniko to official kaniko project at `kaniko-project`.
If you are a part of [Google organization](https://github.com/orgs/google/people), please see skip to [Kaniko Release Process - Google Contributors](https://github.com/GoogleContainerTools/kaniko/blob/master/RELEASE.md#kaniko-release-process-google-contributors)

Non-Google Contributors or users, please follow the following steps:
1. Follow the setup instruction to fork kaniko repository [here](https://github.com/GoogleContainerTools/kaniko/blob/master/DEVELOPMENT.md#getting-started)
2. Run the following `make` commands to build and push Kaniko image to your organization image repository.
  ```shell
   REGISTRY=gcr.io/YOUR-PROJECT make images
   ```
  The above command will build and push all the 3 kaniko images
  * gcr.io/YOUR-PROJECT/executor:latest
  * gcr.io/YOUR-PROJECT/executor:debug
  * gcr.io/YOUR-PROJECT/warmer:latest

3. You can choose tag these images using `docker tag` 
e.g. To tag `gcr.io/YOUR-PROJECT/executor:latest` as `gcr.io/YOUR-PROJECT/executor:v1.6.0self-serve`, run
   ```shell
    docker tag gcr.io/YOUR-PROJECT/executor:latest gcr.io/YOUR-PROJECT/executor:v1.6.0self-serve
   ```
   
Please change all usages of `gcr.io/kaniko-project/executor:latest` to `gcr.io/YOUR-PROJECT/executor:latest` for executor image and so on.
4. Finally, pushed your tagged images via docker. You could also use the Makefile target `push` to push these images like this
  ```shell
   REGISTRY=gcr.io/YOUR-PROJECT make images
  ```

## Kaniko Release Process [Google Contributors]
### Getting write access to the Kaniko Project
In order to kick off kaniko release, you need to write access to Kaniko project.

To get write access, please ping one of the [Kaniko Maintainers](https://github.com/orgs/GoogleContainerTools/teams/kaniko-maintainers/members). 

Once you have the correct access, you can kick off a release.


### Kicking of a release.

1. Create a release PR and update Changelog.

    In order to release a new version of Kaniko, you will need to first

    a. Create a new branch and bump the kaniko version in [Makefile](https://github.com/GoogleContainerTools/kaniko/blob/master/Makefile#L16)


    In most cases, you will need to bump the `VERSION_MINOR` number.
    In case you are doing a patch release for a hot fix, bump the `VERSION_BUILD` number.

    b. Run the [script](https://github.com/GoogleContainerTools/kaniko/blob/master/hack/release.sh) to create release notes.
    ```
    ./hack/release.sh
    Collecting pull request that were merged since the last release: v1.0.0 (2020-08-18 02:53:46 +0000 UTC)
    * change repo string to just string [#1417](https://github.com/GoogleContainerTools/kaniko/pull/1417)
    * Improve --use-new-run help text, update README with missing flags [#1405](https://github.com/GoogleContainerTools/kaniko/pull/1405)
    ...
    Huge thank you for this release towards our contributors: 
    - Alex Szakaly
    - Alexander Sharov
    ```
    Copy the release notes and update the [CHANGELOG.md](https://github.com/GoogleContainerTools/kaniko/blob/master/CHANGELOG.md) at the root of the repository. 

    c. Create a pull request like [this](https://github.com/GoogleContainerTools/kaniko/pull/1388) and get it approved from Kaniko maintainers.

2. Once the PR is approved and merged, create a release tag with name `vX.Y.Z` where
    ```
    X corresponds to VERSION_MAJOR
    Y corresponds to VERSION_MINOR
    Z corresponds to VERSION_BUILD
    ```
    E.g. to release 1.2.0 version of kaniko, please create a tag v1.2.0 like this
    ```
    git pull remote master
    git tag v1.2.0
    git push remote v1.2.0
    ```
3.  Pushing a tag to remote with above naming convention will trigger the Github workflow action defined [here](https://github.com/GoogleContainerTools/kaniko/blob/main/.github/workflows/images.yaml) It takes 20-30 mins for the job to finish and push images to [`kaniko-project`](https://pantheon.corp.google.com/gcr/images/kaniko-project?orgonly=true&project=kaniko-project&supportedpurview=organizationId)
```
gcr.io/kaniko-project/executor:latest
gcr.io/kaniko-project/executor:vX.Y.Z
gcr.io/kaniko-project/executor:debug
gcr.io/kaniko-project/executor:debug-vX.Y.Z
gcr.io/kaniko-project/executor:warmer
gcr.io/kaniko-project/executor:warmer-vX.Y.Z
```
You could verify if the images are published using the `docker pull` command
```
docker pull gcr.io/kaniko-project/executor:vX.Y.Z
docker pull gcr.io/kaniko-project/warmer:vX.Y.Z
```
In case the images are still not published, ping one of the kaniko maintainers and they will provide the cloud build trigger logs.
You can also request read access to the Google `kaniko-project`.

4. Finally, once the images are published, create a release for the newly created [tag](https://github.com/GoogleContainerTools/kaniko/tags) and publish it. 
Summarize the change log to mention, 
- new features added if any
- bug fixes, 
- refactors and 
- documentation changes
