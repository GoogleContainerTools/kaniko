package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	docker_credentials "github.com/docker/docker-credential-helpers/credentials"
)

var ecrHostname = regexp.MustCompile(`^[0-9]+\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`)
var ghcrHostname = regexp.MustCompile(`^ghcr\.io$`)

const (
	defaultScheme     = "https://"
	envPrefix         = "DOCKER"
	envUsernameSuffix = "USR"
	envPasswordSuffix = "PSW"
	envSeparator      = "_"
	envIgnoreLogin    = "IGNORE_DOCKER_LOGIN"
)

type NotSupportedError struct{}

func (m *NotSupportedError) Error() string {
	return "not supported"
}

// Env implements the Docker credentials Helper interface.
type Env struct{}

// Add implements the set verb
func (*Env) Add(*docker_credentials.Credentials) error {
	switch {
	case os.Getenv(envIgnoreLogin) != "":
		return nil
	default:
		return fmt.Errorf("add: %w", &NotSupportedError{})
	}
}

// Delete implements the erase verb
func (*Env) Delete(string) error {
	switch {
	case os.Getenv(envIgnoreLogin) != "":
		return nil
	default:
		return fmt.Errorf("delete: %w", &NotSupportedError{})
	}
}

// List implements the list verb
func (*Env) List() (map[string]string, error) {
	return nil, fmt.Errorf("list: %w", &NotSupportedError{})
}

// Get implements the get verb
func (e *Env) Get(serverURL string) (username string, password string, err error) {
	var (
		hostname string
		ok       bool
	)

	hostname, err = getHostname(serverURL)
	if err != nil {
		return
	}

	if username, password, ok = getEnvCredentials(hostname); ok {
		return
	}

	if ecrHostname.MatchString(hostname) {
		// This is an AWS ECR Docker Registry: <account-id>.dkr.ecr.<region>.amazonaws.com
		username, password, err = getEcrToken()
		return
	}

	if ghcrHostname.MatchString(hostname) {
		// This is a GitHub Container Registry: ghcr.io
		if token, found := os.LookupEnv("GITHUB_TOKEN"); found {
			username = "github"
			password = token
		}
		return
	}

	return
}

func getHostname(serverURL string) (hostname string, err error) {
	var server *url.URL
	server, err = url.Parse(defaultScheme + strings.TrimPrefix(serverURL, defaultScheme))
	if err != nil {
		return
	}

	hostname = server.Hostname()

	return
}

func getEnvVariables(labels []string, offset int) (envUsername, envPassword string) {
	if offset < 0 {
		offset = 0
	} else if offset > len(labels) {
		offset = len(labels)
	}

	envHostname := strings.Join(labels[offset:], envSeparator)
	envUsername = strings.Join([]string{envPrefix, envHostname, envUsernameSuffix}, envSeparator)
	envPassword = strings.Join([]string{envPrefix, envHostname, envPasswordSuffix}, envSeparator)

	return
}

func getEnvCredentials(hostname string) (username, password string, found bool) {
	hostname = strings.ReplaceAll(hostname, "-", "_")
	labels := strings.Split(hostname, ".")

	for i := 0; i <= len(labels); i++ {
		envUsername, envPassword := getEnvVariables(labels, i)

		if username, found = os.LookupEnv(envUsername); found {
			if password, found = os.LookupEnv(envPassword); found {
				break
			}
		}
	}
	return
}

func getEcrToken() (username, password string, err error) {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return
	}

	if roleArn := getRoleArn(cfg.ConfigSources...); roleArn != "" {
		stsSvc := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsSvc, roleArn)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	client := ecr.NewFromConfig(cfg)

	output, err := client.GetAuthorizationToken(ctx, nil)
	if err != nil {
		return
	}
	for _, authData := range output.AuthorizationData {
		// authData.AuthorizationToken is a base64-encoded username:password string,
		// where the username is always expected to be "AWS".
		var tokenBytes []byte
		tokenBytes, err = base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
		if err != nil {
			return
		}
		token := bytes.SplitN(tokenBytes, []byte{':'}, 2)
		username, password = string(token[0]), string(token[1])
	}
	return
}

func getRoleArn(configSources ...interface{}) (roleARN string) {
	for _, x := range configSources {
		switch impl := x.(type) {
		case config.EnvConfig:
			if impl.RoleARN != "" {
				return strings.TrimSpace(impl.RoleARN)
			}
		case config.SharedConfig:
			if impl.RoleARN != "" {
				return strings.TrimSpace(impl.RoleARN)
			}
		}
	}
	return
}
