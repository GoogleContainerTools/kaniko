package storage

import (
	"bytes"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/encryption"
)

// Common test utilities

type meowCrypter struct {
	// only take encryption failures - decrypt failures can happen if the bytes
	// do not have a cat
	encryptFailures map[string]struct{}
}

func (m meowCrypter) Encrypt(orig []byte) (*api.MaybeEncryptedRecord, error) {
	if _, ok := m.encryptFailures[string(orig)]; ok {
		return nil, fmt.Errorf("refusing to encrypt")
	}
	return &api.MaybeEncryptedRecord{
		Algorithm: m.Algorithm(),
		Data:      append(orig, []byte("🐱")...),
	}, nil
}

func (m meowCrypter) Decrypt(orig api.MaybeEncryptedRecord) ([]byte, error) {
	if orig.Algorithm != m.Algorithm() || !bytes.HasSuffix(orig.Data, []byte("🐱")) {
		return nil, fmt.Errorf("not meowcoded")
	}
	return bytes.TrimSuffix(orig.Data, []byte("🐱")), nil
}

func (m meowCrypter) Algorithm() api.MaybeEncryptedRecord_Algorithm {
	return api.MaybeEncryptedRecord_Algorithm(-1)
}

var _ encryption.Encrypter = meowCrypter{}
var _ encryption.Decrypter = meowCrypter{}
