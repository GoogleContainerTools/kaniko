// docker-credentials-env is a Docker credentials helper that reads
// credentials from the process environment.

package main

import (
	docker_credentials "github.com/docker/docker-credential-helpers/credentials"
)

func main() {
	docker_credentials.Serve(&Env{})
}
