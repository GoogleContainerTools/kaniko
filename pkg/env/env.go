package env

import (
	"os"
)

func SetEnvironmentVariables(image string) error {
	//TODO: uncomment following
	os.Setenv("HOME", "/root")
	return os.Setenv("PATH", "/opt/python3.5/bin:/opt/python3.6/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/root")
}
