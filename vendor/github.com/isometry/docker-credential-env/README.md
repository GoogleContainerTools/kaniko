# Docker Credentials from the Environment

A [Docker credential helper](https://docs.docker.com/engine/reference/commandline/login/#credential-helpers) to streamline repository interactions in scenarios where the cacheing of credentials to `~/.docker/config.json` is undesirable, including CI/CD pipelines, or anywhere ephemeral credentials are used.

All OCI registry clients that support `~/.docker/config.json` are supported, including [`oras`](https://oras.land/), [`crane`](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md), [`grype`](https://github.com/anchore/grype), etc.

In addition to handling basic username:password credentials, the credential helper also includes special support for:

* Amazon Elastic Container Registry (ECR) repositories using [standard AWS credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html), including automatic cross-account role assumption.
* [GitHub Packages](https://ghcr.io/) via the common `GITHUB_TOKEN` environment variable.

## Environment Variables

For the docker repository `https://repo.example.com/v1`, the credential helper expects to retrieve credentials from the following environment variables:

* `DOCKER_repo_example_com_USR` containing the repository username
* `DOCKER_repo_example_com_PSW` containing the repository password, token or secret.

If no environment variables for the target repository's FQDN is found, then:

1. The helper will remove DNS labels from the FQDN one-at-a-time from the right, and look again, for example:
`DOCKER_repo_example_com_USR` => `DOCKER_example_com_USR` => `DOCKER_com_USR` => `DOCKER__USR`.
2. If the target repository is a private AWS ECR repository (FQDN matches the regex `^[0-9]+\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`), it will attempt to exchange local AWS credentials (most likely exposed through `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables) for short-lived ECR login credentials, including automatic sts:AssumeRole if `role_arn` is specified (e.g. via `AWS_ROLE_ARN`).

Hyphens within DNS labels are transformed to underscores (`s/-/_/g`) for the purposes of credential lookup.

## Configuration

The `docker-credential-env` binary must be installed to `$PATH`, and is enabled via `~/.docker/config.json`:

* Handle all docker authentication:

  ```json
  {
    "credsStore": "env"
  }
  ```

* Handle docker authentication for specific repositories:

  ```json
  {
    "credHelpers": {
      "artifactory.example.com": "env"
    }
  }
  ```

By default, attempts to explicitly `docker {login,logout}` will generate an error. To ignore these errors, set the environment variable `IGNORE_DOCKER_LOGIN=1`.

## Example Usage

### Jenkins

```groovy
stages {
    stage('Push Image to Artifactory') {
        environment {
            DOCKER_artifactory_example_com = credentials('jenkins.artifactory') // (Vault) Username-Password credential
        }
        steps {
            sh 'docker push artifactory.example.com/example/example-image:1.0'
        }
    }

    stage('Push Image to Docker Hub') {
        environment {
            DOCKER_docker_com = credentials('hub.docker.com') // Username-Password credential, exploiting domain search
        }
        steps {
            sh 'docker push hub.docker.com/example/example-image:1.0'
        }
    }

    stage('Push Image to AWS-ECR') {
        environment {
            // any standard AWS authentication mechanisms are supported
            AWS_ROLE_ARN          = 'arn:aws:iam::123456789:role/jenkins-user' // triggers automatic sts:AssumeRole
            // AWS_CONFIG_FILE    = file('AWS_CONFIG')
            // AWS_PROFILE        = 'jenkins'
            AWS_ACCESS_KEY_ID     = credentials('AWS_ACCESS_KEY_ID') // String credential
            AWS_SECRET_ACCESS_KEY = credentials('AWS_SECRET_ACCESS_KEY') // String credential
        }
        steps {
            sh 'docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/example/example-image:1.0'
        }
    }

      stage('Push Image to GHCR') {
        environment {
            GITHUB_TOKEN = credentials('github') // String credential
        }
        steps {
            sh 'docker push ghcr.io/example/example-image:1.0'
        }
    }

}
```
