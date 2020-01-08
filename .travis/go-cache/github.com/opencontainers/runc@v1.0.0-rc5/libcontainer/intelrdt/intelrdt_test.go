// +build linux

package intelrdt

import (
	"strings"
	"testing"
)

func TestIntelRdtSetL3CacheSchema(t *testing.T) {
	if !IsEnabled() {
		return
	}

	helper := NewIntelRdtTestUtil(t)
	defer helper.cleanup()

	const (
		l3CacheSchemaBefore = "L3:0=f;1=f0"
		l3CacheSchemeAfter  = "L3:0=f0;1=f"
	)

	helper.writeFileContents(map[string]string{
		"schemata": l3CacheSchemaBefore + "\n",
	})

	helper.IntelRdtData.config.IntelRdt.L3CacheSchema = l3CacheSchemeAfter
	intelrdt := &IntelRdtManager{
		Config: helper.IntelRdtData.config,
		Path:   helper.IntelRdtPath,
	}
	if err := intelrdt.Set(helper.IntelRdtData.config); err != nil {
		t.Fatal(err)
	}

	tmpStrings, err := getIntelRdtParamString(helper.IntelRdtPath, "schemata")
	if err != nil {
		t.Fatalf("Failed to parse file 'schemata' - %s", err)
	}
	values := strings.Split(tmpStrings, "\n")
	value := values[0]

	if value != l3CacheSchemeAfter {
		t.Fatal("Got the wrong value, set 'schemata' failed.")
	}
}
