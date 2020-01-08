package container

import (
	"testing"
)

func TestReadUserMappings(t *testing.T) {
	f := `         0     100000       1000
      1000       1000          1
      1001     101001      64535`
	expected := []UserMapping{
		{
			ContainerID: 0,
			HostID:      100000,
			Range:       1000,
		},
		{
			ContainerID: 1000,
			HostID:      1000,
			Range:       1,
		},
		{
			ContainerID: 1001,
			HostID:      101001,
			Range:       64535,
		},
	}

	userNs, mappings, err := readUserMappings(f)
	if err != nil {
		t.Fatal(err)
	}

	if !userNs {
		t.Fatal("expected user namespaces to be true")
	}

	if len(expected) != len(mappings) {
		t.Fatalf("expected length %d got %d", len(expected), len(mappings))
	}
}
