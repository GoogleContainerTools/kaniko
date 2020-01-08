package integration

import (
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Sandbox interface {
	Address() string
	PrintLogs(*testing.T)
	Cmd(...string) *exec.Cmd
	NewRegistry() (string, error)
	Rootless() bool
}

type Worker interface {
	New() (Sandbox, func() error, error)
	Name() string
}

type Test func(*testing.T, Sandbox)

var defaultWorkers []Worker

func register(w Worker) {
	defaultWorkers = append(defaultWorkers, w)
}

func List() []Worker {
	return defaultWorkers
}

func Run(t *testing.T, testCases []Test) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	for _, br := range List() {
		for _, tc := range testCases {
			ok := t.Run(getFunctionName(tc)+"/worker="+br.Name(), func(t *testing.T) {
				sb, close, err := br.New()
				if err != nil {
					if errors.Cause(err) == ErrorRequirements {
						t.Skip(err.Error())
					}
					require.NoError(t, err)
				}
				defer func() {
					assert.NoError(t, close())
					if t.Failed() {
						sb.PrintLogs(t)
					}
				}()
				tc(t, sb)
			})
			require.True(t, ok)
		}
	}
}

func getFunctionName(i interface{}) string {
	fullname := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	dot := strings.LastIndex(fullname, ".") + 1
	return strings.Title(fullname[dot:])
}
