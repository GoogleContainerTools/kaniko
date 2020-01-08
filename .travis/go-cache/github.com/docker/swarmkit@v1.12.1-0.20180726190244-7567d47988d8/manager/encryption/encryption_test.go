package encryption

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	// not providing an encrypter will fail
	msg := []byte("hello again swarmkit")
	_, err := Encrypt(msg, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no encrypter")

	// noop encrypter can encrypt
	encrypted, err := Encrypt(msg, NoopCrypter)
	require.NoError(t, err)

	// not providing a decrypter will fail
	_, err = Decrypt(encrypted, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no decrypter")

	// noop decrypter can decrypt
	decrypted, err := Decrypt(encrypted, NoopCrypter)
	require.NoError(t, err)
	require.Equal(t, msg, decrypted)

	// the default encrypter can produce something the default decrypter can read
	encrypter, decrypter := Defaults([]byte("key"), false)
	encrypted, err = Encrypt(msg, encrypter)
	require.NoError(t, err)
	decrypted, err = Decrypt(encrypted, decrypter)
	require.NoError(t, err)
	require.Equal(t, msg, decrypted)

	// mismatched encrypters and decrypters can't read the content produced by each
	encrypted, err = Encrypt(msg, NoopCrypter)
	require.NoError(t, err)
	_, err = Decrypt(encrypted, decrypter)
	require.Error(t, err)
	require.IsType(t, ErrCannotDecrypt{}, err)

	encrypted, err = Encrypt(msg, encrypter)
	require.NoError(t, err)
	_, err = Decrypt(encrypted, NoopCrypter)
	require.Error(t, err)
	require.IsType(t, ErrCannotDecrypt{}, err)
}

func TestHumanReadable(t *testing.T) {
	// we can produce human readable strings that can then be re-parsed
	key := GenerateSecretKey()
	keyString := HumanReadableKey(key)
	parsedKey, err := ParseHumanReadableKey(keyString)
	require.NoError(t, err)
	require.Equal(t, parsedKey, key)

	// if the prefix is wrong, we can't parse the key
	_, err = ParseHumanReadableKey("A" + keyString)
	require.Error(t, err)

	// With the right prefix, we can't parse if the key isn't base64 encoded
	_, err = ParseHumanReadableKey(humanReadablePrefix + "aaa*aa/")
	require.Error(t, err)

	// Extra padding also fails
	_, err = ParseHumanReadableKey(keyString + "=")
	require.Error(t, err)
}

type bothCrypter interface {
	Decrypter
	Encrypter
}

func TestMultiDecryptor(t *testing.T) {
	crypters := []bothCrypter{
		noopCrypter{},
		NewNACLSecretbox([]byte("key1")),
		NewNACLSecretbox([]byte("key2")),
		NewNACLSecretbox([]byte("key3")),
		NewFernet([]byte("key1")),
		NewFernet([]byte("key2")),
	}
	m := NewMultiDecrypter(
		crypters[0], crypters[1], crypters[2], crypters[4],
		NewMultiDecrypter(crypters[3], crypters[5]),
	)

	for i, c := range crypters {
		plaintext := []byte(fmt.Sprintf("message %d", i))
		ciphertext, err := Encrypt(plaintext, c)
		require.NoError(t, err)
		decrypted, err := Decrypt(ciphertext, m)
		require.NoError(t, err)
		require.Equal(t, plaintext, decrypted)

		// for sanity, make sure the other crypters can't decrypt
		for j, o := range crypters {
			if j == i {
				continue
			}
			_, err := Decrypt(ciphertext, o)
			require.IsType(t, ErrCannotDecrypt{}, err)
		}
	}

	// Test multidecryptor where it does not have a decryptor with the right key
	for _, d := range []MultiDecrypter{m, NewMultiDecrypter()} {
		plaintext := []byte("message")
		ciphertext, err := Encrypt(plaintext, NewNACLSecretbox([]byte("other")))
		require.NoError(t, err)
		_, err = Decrypt(ciphertext, d)
		require.IsType(t, ErrCannotDecrypt{}, err)
	}
}

// The default encrypter/decrypter, if FIPS is not enabled, is NACLSecretBox.
// However, it can decrypt using all other supported algorithms.  If FIPS is
// enabled, the encrypter/decrypter is Fernet only, because FIPS only permits
// (given the algorithms swarmkit supports) AES-128-CBC
func TestDefaults(t *testing.T) {
	plaintext := []byte("my message")

	// encrypt something without FIPS enabled
	c, d := Defaults([]byte("key"), false)
	ciphertext, err := Encrypt(plaintext, c)
	require.NoError(t, err)
	decrypted, err := Decrypt(ciphertext, d)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)

	// with fips enabled, defaults should return a fernet encrypter
	// and a decrypter that can't decrypt nacl
	c, d = Defaults([]byte("key"), true)
	_, err = Decrypt(ciphertext, d)
	require.Error(t, err)
	ciphertext, err = Encrypt(plaintext, c)
	require.NoError(t, err)
	decrypted, err = Decrypt(ciphertext, d)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)

	// without FIPS, and ensure we can decrypt the previous ciphertext
	// (encrypted with fernet) with the decrypter returned by defaults
	_, d = Defaults([]byte("key"), false)
	decrypted, err = Decrypt(ciphertext, d)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}
