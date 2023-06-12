// +build tools

package tools

//  dependencies https://github.com/golang/go/issues/48332
import (
	_ "github.com/GoogleCloudPlatform/docker-credential-gcr"
	_ "github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cli/docker-credential-ecr-login"
	_ "github.com/chrismellard/docker-credential-acr-env"
)
