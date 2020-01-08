package encryption

import (
	cryptorand "crypto/rand"
	"io"
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/require"
)

// Using the same key to encrypt the same message, this encrypter produces two
// different ciphertexts because it produces two different nonces.  Both
// of these can be decrypted into the same data though.
func TestNACLSecretbox(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)
	keyCopy := make([]byte, 32)
	copy(key, keyCopy)

	crypter1 := NewNACLSecretbox(key)
	crypter2 := NewNACLSecretbox(keyCopy)
	data := []byte("Hello again world")

	er1, err := crypter1.Encrypt(data)
	require.NoError(t, err)

	er2, err := crypter1.Encrypt(data)
	require.NoError(t, err)

	require.NotEqual(t, er1.Data, er2.Data)
	require.NotEmpty(t, er1.Nonce)
	require.NotEmpty(t, er2.Nonce)

	// both crypters can decrypt the other's text
	for _, decrypter := range []Decrypter{crypter1, crypter2} {
		for _, record := range []*api.MaybeEncryptedRecord{er1, er2} {
			result, err := decrypter.Decrypt(*record)
			require.NoError(t, err)
			require.Equal(t, data, result)
		}
	}
}

func TestNACLSecretboxInvalidAlgorithm(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)

	crypter := NewNACLSecretbox(key)
	er, err := crypter.Encrypt([]byte("Hello again world"))
	require.NoError(t, err)
	er.Algorithm = api.MaybeEncryptedRecord_NotEncrypted

	_, err = crypter.Decrypt(*er)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a NACL secretbox")
}

func TestNACLSecretboxCannotDecryptWithoutRightKey(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)

	crypter := NewNACLSecretbox(key)
	er, err := crypter.Encrypt([]byte("Hello again world"))
	require.NoError(t, err)

	crypter = NewNACLSecretbox([]byte{})
	_, err = crypter.Decrypt(*er)
	require.Error(t, err)
}

func TestNACLSecretboxInvalidNonce(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)

	crypter := NewNACLSecretbox(key)
	er, err := crypter.Encrypt([]byte("Hello again world"))
	require.NoError(t, err)
	er.Nonce = er.Nonce[:20]

	_, err = crypter.Decrypt(*er)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid nonce size")
}
