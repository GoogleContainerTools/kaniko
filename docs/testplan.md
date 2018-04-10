# Kaniko Test Plan


## What do we want to test?



*   The basic premise is to build the same Dockerfile, both through docker then through kaniko, then compare the resulting images. The goal is for the images to be identical, for some definition of identical, defined below.


## What are the testing priorities?



    *   Speed / Parallelization
        *   We can build all the docker-built images and the kaniko-built images in parallel using Argo. Once everything is built, we can compare each respective image using container-diff (see what's "good enough" below). Once the newest container-diff release goes out, we'll be able to compare metadata as well and will no longer need to use container-structure-test, and the necessity to differentiate between types of tests will no longer exist.
    *   Clear error messages
        *   Currently each test is an argo build test that runs a container-diff check after both images are built. If one of the test fails, the entire test halts and we get a cryptic Argo error message. By building everything first then comparing, we're in charge of what error gets displayed on failure and we can run all the tests instead of halting at first failure.
    *   Easy of use / Ability to read and add tests
        *   The goal is to able to add a test to the suite by just adding a Dockerfile to a directory. The test will scan the directory for Dockerfiles and generate a test for each file. It will assume container-diff needs to return nothing by default, which can be overridden if needed (e.g. config_test_run.json). We should also be able to test any individual Dockerfile. 
    *   Thorough testing of each command 
        *   Each command currently implemented by Kaniko should have its own Dockerfile (and therefore its own test). As bugs come in and get fixed, we should be able to just add a new Dockerfile testing the bug and watch it pass. 


## What's "good enough?"



*   Container-diff doesn't give us enough granularity since it only checks the resulting file system and metadata, without checking the layers themselves. If we add the ability to check each layer of both images side-by-side to container-diff, we could determine whether they are the same of not with confidence,
*   We can potentially verify checksums of either the layers or the entire image as well.


## How are we going to do this?



*   Our current integration tests use` go test `to generate an Argo yaml. This yaml loops through each Dockerfile, builds it with docker and with Kaniko in parallel, then uses container-diff to test the two resulting images. If container-diff returns something unexpected, Argo exits and the entire test halts. 
*   The plan is to use go subtests.
    *   Setup will be building all the images at once (or if a specific test is requested, just build the two images for that specific Dockerfile). 
    *   Then for each Dockerfile, invoke container-diff. If that succeeds, unpack each image tarball (this means we either need to push both images up to GCR or to docker save them at build time. I don't see a specific advantage to either) and inspect the layers.
        *   If we pass, add it to the list of passing tests.
        *   If we fail, specify which Dockerfile failed, and exactly what differs.
    *   Teardown will be deleting all generated images.


## What about testing on-cluster builds?



*   We should be able to have a test Kubernetes cluster that can spin up pods for each Dockerfile. This will achieve everything we want for the tests while testing our main Kaniko use case.


## Should we use this in CI?



*   In order to reliably use this in Travis (or any other CI system), we need to separate Dockerfiles that result in non-reproducible image into a separate smoke test category. They need to be tested, but having them in CI doesn't make sense.