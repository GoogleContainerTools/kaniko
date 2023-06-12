# Contributing to docker-credential-gcr

## <a name="cla"></a> Sign the CLA

Contributions to any Google project must be accompanied by a Contributor License Agreement.  This is not a copyright **assignment**, it simply gives Google permission to use and redistribute your contributions as part of the project. Head over to <https://cla.developers.google.com/> to see your current agreements on file or to sign a new one. You may make a pull request before you have signed a CLA, but the request will not be merged until you have.

You generally only need to submit a CLA once, so if you've already submitted one (even if it was for a different project), you probably don't need to do it again.

## <a name="submit"></a> Submission Guidelines
### Submitting a Pull Request
Before you submit your pull request consider the following guidelines:

* Search [GitHub](https://github.com/GoogleCloudPlatform/docker-credential-gcr/pulls) for an open or closed Pull Request that relates to your submission.
* Please sign our [Contributor License Agreement (CLA)](#cla) before sending pull requests. We cannot accept code without this.
* Make your changes in a new git branch:

     ```shell
     git checkout -b my-fix-branch master
     ```

* Create your patch, **including appropriate test cases**.
* Follow our [Coding Rules](#rules).
* Run the full test suite.

     ```shell
     go build
     go test -timeout 10s -v ./...
     ```
* Commit your changes using a descriptive commit message.

     ```shell
     git commit -a -m "omg y u bad @ coding"
     ```
  Note: the optional commit `-a` command line option will automatically "add" and "rm" edited files.

* Push your branch to GitHub:

    ```shell
    git push origin my-fix-branch
    ```

* In GitHub, send a pull request to `docker-credential-gcr:master`.
* If we suggest changes then:
  * Make the required updates.
  * Re-run the test suite to ensure tests are still passing.
  * Commit your changes to your branch (e.g. `my-fix-branch`).
  * Push the changes to your GitHub repository (this will update your Pull Request).

If the PR gets too outdated we may ask you to rebase and force push to update the PR:

```shell
git rebase master -i
git push origin my-fix-branch -f
```

*WARNING. Squashing or reverting commits and forced push thereafter may remove GitHub comments on code that were previously made by you and others in your commits.*

That's it! Thank you for your contribution!

#### After your pull request is merged

After your pull request is merged, you can safely delete your branch and pull the changes from the main (upstream) repository:

* Delete the remote branch on GitHub either through the GitHub web UI or your local shell as follows:

    ```shell
    git push origin --delete my-fix-branch
    ```

* Check out the master branch:

    ```shell
    git checkout master -f
    ```

* Delete the local branch:

    ```shell
    git branch -D my-fix-branch
    ```

* Update your master with the latest upstream version:

    ```shell
    git pull --ff upstream master
    ```

## <a name="rules"></a> Coding Rules

* Go source code should follow the conventions given in [Effective Go](https://golang.org/doc/effective_go.html).
* Source files must be formatted with `gofmt` and updated with `go fix` before submission.

    ```shell
    go fmt
    go fix
    ```
* Source files should be inspected by `go vet`. Since there may be false positives with both, ignored warnings require justification but won't necessarily block changes.

    ```shell
    go vet
    ```
