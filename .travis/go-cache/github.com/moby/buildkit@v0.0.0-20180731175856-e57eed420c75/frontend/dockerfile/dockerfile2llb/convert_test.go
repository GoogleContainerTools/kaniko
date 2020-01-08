package dockerfile2llb

import (
	"testing"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/stretchr/testify/assert"
)

func TestDockerfileParsing(t *testing.T) {
	t.Parallel()
	df := `FROM busybox
ENV FOO bar
COPY f1 f2 /sub/
RUN ls -l
`
	_, _, err := Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{})
	assert.NoError(t, err)

	df = `FROM busybox AS foo
ENV FOO bar
FROM foo
COPY --from=foo f1 /
COPY --from=0 f2 /
	`
	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{})
	assert.NoError(t, err)

	df = `FROM busybox AS foo
ENV FOO bar
FROM foo
COPY --from=foo f1 /
COPY --from=0 f2 /
	`
	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{
		Target: "Foo",
	})
	assert.NoError(t, err)

	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{
		Target: "nosuch",
	})
	assert.Error(t, err)

	df = `FROM busybox
	ADD http://github.com/moby/buildkit/blob/master/README.md /
		`
	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{})
	assert.NoError(t, err)

	df = `FROM busybox
	COPY http://github.com/moby/buildkit/blob/master/README.md /
		`
	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{})
	assert.EqualError(t, err, "source can't be a URL for COPY")

	df = `FROM "" AS foo`
	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{})
	assert.Error(t, err)

	df = `FROM ${BLANK} AS foo`
	_, _, err = Dockerfile2LLB(appcontext.Context(), []byte(df), ConvertOpt{})
	assert.Error(t, err)
}

func TestAddEnv(t *testing.T) {
	// k exists in env as key
	// override = true
	env := []string{"key1=val1", "key2=val2"}
	result := addEnv(env, "key1", "value1")
	assert.Equal(t, []string{"key1=value1", "key2=val2"}, result)

	// k does not exist in env as key
	// override = true
	env = []string{"key1=val1", "key2=val2"}
	result = addEnv(env, "key3", "val3")
	assert.Equal(t, []string{"key1=val1", "key2=val2", "key3=val3"}, result)

	// env has same keys
	// override = true
	env = []string{"key1=val1", "key1=val2"}
	result = addEnv(env, "key1", "value1")
	assert.Equal(t, []string{"key1=value1", "key1=val2"}, result)

	// k matches with key only string in env
	// override = true
	env = []string{"key1=val1", "key2=val2", "key3"}
	result = addEnv(env, "key3", "val3")
	assert.Equal(t, []string{"key1=val1", "key2=val2", "key3=val3"}, result)
}

func TestParseKeyValue(t *testing.T) {
	k, v := parseKeyValue("key=val")
	assert.Equal(t, "key", k)
	assert.Equal(t, "val", v)

	k, v = parseKeyValue("key=")
	assert.Equal(t, "key", k)
	assert.Equal(t, "", v)

	k, v = parseKeyValue("key")
	assert.Equal(t, "key", k)
	assert.Equal(t, "", v)
}

func TestToEnvList(t *testing.T) {
	// args has no duplicated key with env
	v := "val2"
	args := []instructions.KeyValuePairOptional{{Key: "key2", Value: &v}}
	env := []string{"key1=val1"}
	resutl := toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": "val1", "key2": "val2"}, resutl)

	// value of args is nil
	args = []instructions.KeyValuePairOptional{{Key: "key2", Value: nil}}
	env = []string{"key1=val1"}
	resutl = toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": "val1", "key2": ""}, resutl)

	// args has duplicated key with env
	v = "val2"
	args = []instructions.KeyValuePairOptional{{Key: "key1", Value: &v}}
	env = []string{"key1=val1"}
	resutl = toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": "val1"}, resutl)

	v = "val2"
	args = []instructions.KeyValuePairOptional{{Key: "key1", Value: &v}}
	env = []string{"key1="}
	resutl = toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": ""}, resutl)

	v = "val2"
	args = []instructions.KeyValuePairOptional{{Key: "key1", Value: &v}}
	env = []string{"key1"}
	resutl = toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": ""}, resutl)

	// env has duplicated keys
	v = "val2"
	args = []instructions.KeyValuePairOptional{{Key: "key2", Value: &v}}
	env = []string{"key1=val1", "key1=val1_2"}
	resutl = toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": "val1", "key2": "val2"}, resutl)

	// args has duplicated keys
	v1 := "v1"
	v2 := "v2"
	args = []instructions.KeyValuePairOptional{{Key: "key2", Value: &v1}, {Key: "key2", Value: &v2}}
	env = []string{"key1=val1"}
	resutl = toEnvMap(args, env)
	assert.Equal(t, map[string]string{"key1": "val1", "key2": "v1"}, resutl)
}
