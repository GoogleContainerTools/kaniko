package controlapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func createConfigSpec(name string, data []byte, labels map[string]string) *api.ConfigSpec {
	return &api.ConfigSpec{
		Annotations: api.Annotations{Name: name, Labels: labels},
		Data:        data,
	}
}

func TestValidateConfigSpec(t *testing.T) {
	type BadServiceSpec struct {
		spec *api.ServiceSpec
		c    codes.Code
	}

	for _, badName := range []string{
		"",
		".",
		"-",
		"_",
		".name",
		"name.",
		"-name",
		"name-",
		"_name",
		"name_",
		"/a",
		"a/",
		"a/b",
		"..",
		"../a",
		"a/..",
		"withexclamation!",
		"with space",
		"with\nnewline",
		"with@splat",
		"with:colon",
		"with;semicolon",
		"snowman☃",
		strings.Repeat("a", 65),
	} {
		err := validateConfigSpec(createConfigSpec(badName, []byte("valid config"), nil))
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))
	}

	for _, badSpec := range []*api.ConfigSpec{
		nil,
		createConfigSpec("validName", nil, nil),
	} {
		err := validateConfigSpec(badSpec)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))
	}

	for _, goodName := range []string{
		"0",
		"a",
		"A",
		"name-with--dashes",
		"name.with..dots",
		"name_with__underscores",
		"name.with-all_special",
		"02624name035with1699numbers015125",
		strings.Repeat("a", 64),
	} {
		err := validateConfigSpec(createConfigSpec(goodName, []byte("valid config"), nil))
		assert.NoError(t, err)
	}

	for _, good := range []*api.ConfigSpec{
		createConfigSpec("validName", []byte("☃\n\t\r\x00 dg09236l;kajdgaj5%#9836[Q@!$]"), nil),
		createConfigSpec("validName", []byte("valid config"), nil),
		createConfigSpec("createName", make([]byte, 1), nil), // 1 byte
	} {
		err := validateConfigSpec(good)
		assert.NoError(t, err)
	}
}

func TestCreateConfig(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// ---- creating a config with an invalid spec fails, thus checking that CreateConfig validates the spec ----
	_, err := ts.Client.CreateConfig(context.Background(), &api.CreateConfigRequest{Spec: createConfigSpec("", nil, nil)})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))

	// ---- creating a config with a valid spec succeeds, and returns a config that reflects the config in the store
	// exactly
	data := []byte("config")
	creationSpec := createConfigSpec("name", data, nil)
	validSpecRequest := api.CreateConfigRequest{Spec: creationSpec}

	resp, err := ts.Client.CreateConfig(context.Background(), &validSpecRequest)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Config)
	assert.Equal(t, *creationSpec, resp.Config.Spec)

	// for sanity, check that the stored config still has the config data
	var storedConfig *api.Config
	ts.Store.View(func(tx store.ReadTx) {
		storedConfig = store.GetConfig(tx, resp.Config.ID)
	})
	assert.NotNil(t, storedConfig)
	assert.Equal(t, data, storedConfig.Spec.Data)

	// ---- creating a config with the same name, even if it's the exact same spec, fails due to a name conflict ----
	_, err = ts.Client.CreateConfig(context.Background(), &validSpecRequest)
	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, grpc.Code(err), grpc.ErrorDesc(err))
}

func TestGetConfig(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// ---- getting a config without providing an ID results in an InvalidArgument ----
	_, err := ts.Client.GetConfig(context.Background(), &api.GetConfigRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))

	// ---- getting a non-existent config fails with NotFound ----
	_, err = ts.Client.GetConfig(context.Background(), &api.GetConfigRequest{ConfigID: "12345"})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err), grpc.ErrorDesc(err))

	// ---- getting an existing config returns the config ----
	config := configFromConfigSpec(createConfigSpec("name", []byte("data"), nil))
	err = ts.Store.Update(func(tx store.Tx) error {
		return store.CreateConfig(tx, config)
	})
	assert.NoError(t, err)

	resp, err := ts.Client.GetConfig(context.Background(), &api.GetConfigRequest{ConfigID: config.ID})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Config)
	assert.Equal(t, config, resp.Config)
}

func TestUpdateConfig(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// Add a config to the store to update
	config := configFromConfigSpec(createConfigSpec("name", []byte("data"), map[string]string{"mod2": "0", "mod4": "0"}))
	err := ts.Store.Update(func(tx store.Tx) error {
		return store.CreateConfig(tx, config)
	})
	assert.NoError(t, err)

	// updating a config without providing an ID results in an InvalidArgument
	_, err = ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))

	// getting a non-existent config fails with NotFound
	_, err = ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{ConfigID: "1234adsaa", ConfigVersion: &api.Version{Index: 1}})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err), grpc.ErrorDesc(err))

	// updating an existing config's data returns an error
	config.Spec.Data = []byte{1}
	resp, err := ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{
		ConfigID:      config.ID,
		Spec:          &config.Spec,
		ConfigVersion: &config.Meta.Version,
	})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))

	// updating an existing config's Name returns an error
	config.Spec.Data = nil
	config.Spec.Annotations.Name = "AnotherName"
	resp, err = ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{
		ConfigID:      config.ID,
		Spec:          &config.Spec,
		ConfigVersion: &config.Meta.Version,
	})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))

	// updating the config with the original spec succeeds
	config.Spec.Data = []byte("data")
	config.Spec.Annotations.Name = "name"
	assert.NotNil(t, config.Spec.Data)
	resp, err = ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{
		ConfigID:      config.ID,
		Spec:          &config.Spec,
		ConfigVersion: &config.Meta.Version,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Config)

	// updating an existing config's labels returns the config
	newLabels := map[string]string{"mod2": "0", "mod4": "0", "mod6": "0"}
	config.Spec.Annotations.Labels = newLabels
	config.Spec.Data = nil
	resp, err = ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{
		ConfigID:      config.ID,
		Spec:          &config.Spec,
		ConfigVersion: &resp.Config.Meta.Version,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Config)
	assert.Equal(t, []byte("data"), resp.Config.Spec.Data)
	assert.Equal(t, resp.Config.Spec.Annotations.Labels, newLabels)

	// updating a config with nil data and correct name succeeds again
	config.Spec.Data = nil
	config.Spec.Annotations.Name = "name"
	resp, err = ts.Client.UpdateConfig(context.Background(), &api.UpdateConfigRequest{
		ConfigID:      config.ID,
		Spec:          &config.Spec,
		ConfigVersion: &resp.Config.Meta.Version,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Config)
	assert.Equal(t, []byte("data"), resp.Config.Spec.Data)
	assert.Equal(t, resp.Config.Spec.Annotations.Labels, newLabels)
}

func TestRemoveUnusedConfig(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// removing a config without providing an ID results in an InvalidArgument
	_, err := ts.Client.RemoveConfig(context.Background(), &api.RemoveConfigRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))

	// removing a config that exists succeeds
	config := configFromConfigSpec(createConfigSpec("name", []byte("data"), nil))
	err = ts.Store.Update(func(tx store.Tx) error {
		return store.CreateConfig(tx, config)
	})
	assert.NoError(t, err)

	resp, err := ts.Client.RemoveConfig(context.Background(), &api.RemoveConfigRequest{ConfigID: config.ID})
	assert.NoError(t, err)
	assert.Equal(t, api.RemoveConfigResponse{}, *resp)

	// ---- it was really removed because attempting to remove it again fails with a NotFound ----
	_, err = ts.Client.RemoveConfig(context.Background(), &api.RemoveConfigRequest{ConfigID: config.ID})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err), grpc.ErrorDesc(err))

}

func TestRemoveUsedConfig(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// Create two configs
	data := []byte("config")
	creationSpec := createConfigSpec("configID1", data, nil)
	resp, err := ts.Client.CreateConfig(context.Background(), &api.CreateConfigRequest{Spec: creationSpec})
	assert.NoError(t, err)
	creationSpec2 := createConfigSpec("configID2", data, nil)
	resp2, err := ts.Client.CreateConfig(context.Background(), &api.CreateConfigRequest{Spec: creationSpec2})
	assert.NoError(t, err)

	// Create a service that uses a config
	service := createSpec("service1", "image", 1)
	configRefs := []*api.ConfigReference{
		{
			ConfigName: resp.Config.Spec.Annotations.Name,
			ConfigID:   resp.Config.ID,
			Target: &api.ConfigReference_File{
				File: &api.FileTarget{
					Name: "target.txt",
				},
			},
		},
	}
	service.Task.GetContainer().Configs = configRefs
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: service})
	assert.NoError(t, err)

	service2 := createSpec("service2", "image", 1)
	service2.Task.GetContainer().Configs = configRefs
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: service2})
	assert.NoError(t, err)

	// removing a config that exists but is in use fails
	_, err = ts.Client.RemoveConfig(context.Background(), &api.RemoveConfigRequest{ConfigID: resp.Config.ID})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err), grpc.ErrorDesc(err))
	assert.Regexp(t, "service[1-2], service[1-2]", grpc.ErrorDesc(err))

	// removing a config that exists but is not in use succeeds
	_, err = ts.Client.RemoveConfig(context.Background(), &api.RemoveConfigRequest{ConfigID: resp2.Config.ID})
	assert.NoError(t, err)

	// it was really removed because attempting to remove it again fails with a NotFound
	_, err = ts.Client.RemoveConfig(context.Background(), &api.RemoveConfigRequest{ConfigID: resp2.Config.ID})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err), grpc.ErrorDesc(err))
}

func TestListConfigs(t *testing.T) {
	s := newTestServer(t)

	listConfigs := func(req *api.ListConfigsRequest) map[string]*api.Config {
		resp, err := s.Client.ListConfigs(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		byName := make(map[string]*api.Config)
		for _, config := range resp.Configs {
			byName[config.Spec.Annotations.Name] = config
		}
		return byName
	}

	// ---- Listing configs when there are no configs returns an empty list but no error ----
	result := listConfigs(&api.ListConfigsRequest{})
	assert.Len(t, result, 0)

	// ---- Create a bunch of configs in the store so we can test filtering ----
	allListableNames := []string{"aaa", "aab", "abc", "bbb", "bac", "bbc", "ccc", "cac", "cbc", "ddd"}
	configNamesToID := make(map[string]string)
	for i, configName := range allListableNames {
		config := configFromConfigSpec(createConfigSpec(configName, []byte("config"), map[string]string{
			"mod2": fmt.Sprintf("%d", i%2),
			"mod4": fmt.Sprintf("%d", i%4),
		}))
		err := s.Store.Update(func(tx store.Tx) error {
			return store.CreateConfig(tx, config)
		})
		assert.NoError(t, err)
		configNamesToID[configName] = config.ID
	}

	// ---- build up our list of expectations for what configs get filtered ----

	type listTestCase struct {
		desc     string
		expected []string
		filter   *api.ListConfigsRequest_Filters
	}

	listConfigTestCases := []listTestCase{
		{
			desc:     "no filter: all the available configs are returned",
			expected: allListableNames,
			filter:   nil,
		},
		{
			desc:     "searching for something that doesn't match returns an empty list",
			expected: nil,
			filter:   &api.ListConfigsRequest_Filters{Names: []string{"aa"}},
		},
		{
			desc:     "multiple name filters are or-ed together",
			expected: []string{"aaa", "bbb", "ccc"},
			filter:   &api.ListConfigsRequest_Filters{Names: []string{"aaa", "bbb", "ccc"}},
		},
		{
			desc:     "multiple name prefix filters are or-ed together",
			expected: []string{"aaa", "aab", "bbb", "bbc"},
			filter:   &api.ListConfigsRequest_Filters{NamePrefixes: []string{"aa", "bb"}},
		},
		{
			desc:     "multiple ID prefix filters are or-ed together",
			expected: []string{"aaa", "bbb"},
			filter: &api.ListConfigsRequest_Filters{IDPrefixes: []string{
				configNamesToID["aaa"], configNamesToID["bbb"]},
			},
		},
		{
			desc:     "name prefix, name, and ID prefix filters are or-ed together",
			expected: []string{"aaa", "aab", "bbb", "bbc", "ccc", "ddd"},
			filter: &api.ListConfigsRequest_Filters{
				Names:        []string{"aaa", "ccc"},
				NamePrefixes: []string{"aa", "bb"},
				IDPrefixes:   []string{configNamesToID["aaa"], configNamesToID["ddd"]},
			},
		},
		{
			desc:     "all labels in the label map must be matched",
			expected: []string{allListableNames[0], allListableNames[4], allListableNames[8]},
			filter: &api.ListConfigsRequest_Filters{
				Labels: map[string]string{
					"mod2": "0",
					"mod4": "0",
				},
			},
		},
		{
			desc: "name prefix, name, and ID prefix filters are or-ed together, but the results must match all labels in the label map",
			// + indicates that these would be selected with the name/id/prefix filtering, and 0/1 at the end indicate the mod2 value:
			// +"aaa"0, +"aab"1, "abc"0, +"bbb"1, "bac"0, +"bbc"1, +"ccc"0, "cac"1, "cbc"0, +"ddd"1
			expected: []string{"aaa", "ccc"},
			filter: &api.ListConfigsRequest_Filters{
				Names:        []string{"aaa", "ccc"},
				NamePrefixes: []string{"aa", "bb"},
				IDPrefixes:   []string{configNamesToID["aaa"], configNamesToID["ddd"]},
				Labels: map[string]string{
					"mod2": "0",
				},
			},
		},
	}

	// ---- run the filter tests ----

	for _, expectation := range listConfigTestCases {
		result := listConfigs(&api.ListConfigsRequest{Filters: expectation.filter})
		assert.Len(t, result, len(expectation.expected), expectation.desc)
		for _, name := range expectation.expected {
			assert.Contains(t, result, name, expectation.desc)
			assert.NotNil(t, result[name], expectation.desc)
			assert.Equal(t, configNamesToID[name], result[name].ID, expectation.desc)
			assert.NotNil(t, result[name].Spec.Data)
		}
	}
}
