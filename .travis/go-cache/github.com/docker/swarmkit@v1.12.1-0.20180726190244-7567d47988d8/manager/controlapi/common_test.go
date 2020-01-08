package controlapi

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestValidateAnnotations(t *testing.T) {
	err := validateAnnotations(api.Annotations{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	for _, good := range []api.Annotations{
		{Name: "name"},
		{Name: "n-me"},
		{Name: "n_me"},
		{Name: "n-m-e"},
		{Name: "n--d"},
	} {
		err := validateAnnotations(good)
		assert.NoError(t, err, "string: "+good.Name)
	}

	for _, bad := range []api.Annotations{
		{Name: "_nam"},
		{Name: ".nam"},
		{Name: "-nam"},
		{Name: "nam-"},
		{Name: "n/me"},
		{Name: "n&me"},
		{Name: "////"},
	} {
		err := validateAnnotations(bad)
		assert.Error(t, err, "string: "+bad.Name)
	}
}
