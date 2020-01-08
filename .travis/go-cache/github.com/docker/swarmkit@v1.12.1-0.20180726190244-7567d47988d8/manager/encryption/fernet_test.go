package encryption

import (
	cryptorand "crypto/rand"
	"io"
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/require"
)

// Using the same key to encrypt the same message, this encrypter produces two
// different ciphertexts because the underlying algorithm uses different IVs.
// Both of these can be decrypted into the same data though.
func TestFernet(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)
	keyCopy := make([]byte, 32)
	copy(key, keyCopy)

	crypter1 := NewFernet(key)
	crypter2 := NewFernet(keyCopy)
	data := []byte("Hello again world")

	er1, err := crypter1.Encrypt(data)
	require.NoError(t, err)

	er2, err := crypter2.Encrypt(data)
	require.NoError(t, err)

	require.NotEqual(t, er1.Data, er2.Data)
	require.Empty(t, er1.Nonce)
	require.Empty(t, er2.Nonce)

	// it doesn't matter what the nonce is, it's ignored
	_, err = io.ReadFull(cryptorand.Reader, er1.Nonce)
	require.NoError(t, err)

	// both crypters can decrypt the other's text
	for i, decrypter := range []Decrypter{crypter1, crypter2} {
		for j, record := range []*api.MaybeEncryptedRecord{er1, er2} {
			result, err := decrypter.Decrypt(*record)
			require.NoError(t, err, "error decrypting ciphertext produced by cryptor %d using cryptor %d", j+1, i+1)
			require.Equal(t, data, result)
		}
	}
}

func TestFernetInvalidAlgorithm(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)

	crypter := NewFernet(key)
	er, err := crypter.Encrypt([]byte("Hello again world"))
	require.NoError(t, err)
	er.Algorithm = api.MaybeEncryptedRecord_NotEncrypted

	_, err = crypter.Decrypt(*er)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a Fernet message")
}

func TestFernetCannotDecryptWithoutRightKey(t *testing.T) {
	key := make([]byte, 32)
	_, err := io.ReadFull(cryptorand.Reader, key)
	require.NoError(t, err)

	crypter := NewFernet(key)
	er, err := crypter.Encrypt([]byte("Hello again world"))
	require.NoError(t, err)

	crypter = NewFernet([]byte{})
	_, err = crypter.Decrypt(*er)
	require.Error(t, err)
}
