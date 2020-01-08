package pkcs8

import (
	"encoding/pem"
	"testing"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ecKeyPEM = `-----BEGIN EC PRIVATE KEY-----
MGgCAQEEHAJNi85auxiK2BUfAPoXgYfWt7k57TkK1ng9SCSgBwYFK4EEACGhPAM6
AARNGXPlfrrZHwxfe1yhNNmITamxlJwxA2jpf/qAf4vnE87UZNylsC3xT5Quf/hL
kGCr5YiADxsjiA==
-----END EC PRIVATE KEY-----
`
	decryptedPEM = `-----BEGIN PRIVATE KEY-----
MHgCAQAwEAYHKoZIzj0CAQYFK4EEACEEYTBfAgEBBBwCTYvOWrsYitgVHwD6F4GH
1re5Oe05CtZ4PUgkoTwDOgAETRlz5X662R8MX3tcoTTZiE2psZScMQNo6X/6gH+L
5xPO1GTcpbAt8U+ULn/4S5Bgq+WIgA8bI4g=
-----END PRIVATE KEY-----
`
	encryptedPEM = `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHOMEkGCSqGSIb3DQEFDTA8MBsGCSqGSIb3DQEFDDAOBAiGRncJ5A+72AICCAAw
HQYJYIZIAWUDBAEqBBA0iGGDrKda4SbsQlW8hgiOBIGA1rDEtNqghfQ+8AtdB7kY
US05ElIO2ooXviNo0M36Shltv+1ntd/Qxn+El1B+0BT8MngB8yBV6oFach1dfKvR
PkeX/+bOnd1WTKMx3IPNMWxbA9YPTeoaObaKI7awvI03o51HLd+a5BuHJ55N2CX4
aMbljbOLAjpZS3/VnQteab4=
-----END ENCRYPTED PRIVATE KEY-----
`
	encryptedPEMInvalidPadding = `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHOMEkGCSqGSIb3DQEFDTA8MBsGCSqGSIb3DQEFDDAOBAjxk6v6kjceLAICCAAw
HQYJYIZIAWUDBAEqBBBVCqGMzL53rwf6Bv4OEPeJBIGAEuEUhjZd/d1BEbntAoZU
3cCB6ewYMqj97p6MncR1EFq+a26R/ehoCZg7O2L5AJrZK8K6UuZG8HxpZkraS5Mh
L5dg6PPGclig3Xn1sCPUmHi13x+DPISBuUdkQEep5lEpqxLSRQerllbXmhaTznAk
aqc20eq8ndE9DjZ7gDPnslY=
-----END ENCRYPTED PRIVATE KEY-----`
)

func TestIsEncryptedPEMBlock(t *testing.T) {
	decryptedPEMBlock, _ := pem.Decode([]byte(decryptedPEM))
	encryptedPEMBlock, _ := pem.Decode([]byte(encryptedPEM))

	assert.False(t, IsEncryptedPEMBlock(decryptedPEMBlock))
	assert.True(t, IsEncryptedPEMBlock(encryptedPEMBlock))
}

func TestDecryptPEMBlock(t *testing.T) {
	expectedBlock, _ := pem.Decode([]byte(decryptedPEM))
	block, _ := pem.Decode([]byte(encryptedPEM))

	_, err := DecryptPEMBlock(block, []byte("pony"))
	require.EqualError(t, err, "pkcs8: incorrect password")

	decryptedDer, err := DecryptPEMBlock(block, []byte("ponies"))
	require.NoError(t, err)
	require.Equal(t, expectedBlock.Bytes, decryptedDer)

	// Try to decrypt an already decrypted key
	decryptedKeyBlock, _ := pem.Decode([]byte(decryptedPEM))
	_, err = DecryptPEMBlock(decryptedKeyBlock, []byte("ponies"))
	require.Error(t, err)

	// Decrypt a key with 32bit padding length
	invalidPadLenKeyBlock, _ := pem.Decode([]byte(encryptedPEMInvalidPadding))
	_, err = DecryptPEMBlock(invalidPadLenKeyBlock, []byte("poonies"))
	require.EqualError(t, err, "pkcs8: invalid padding size")
}

func TestEncryptPEMBlock(t *testing.T) {
	block, _ := pem.Decode([]byte(decryptedPEM))
	encryptedBlock, err := EncryptPEMBlock(block.Bytes, []byte("knock knock"))
	require.NoError(t, err)

	// Try to decrypt the same encrypted block
	_, err = DecryptPEMBlock(encryptedBlock, []byte("hey there"))
	require.Error(t, err)

	decryptedDer, err := DecryptPEMBlock(encryptedBlock, []byte("knock knock"))
	require.NoError(t, err)
	require.Equal(t, block.Bytes, decryptedDer)
}

func TestParsePrivateKeyPEMWithPassword(t *testing.T) {
	_, err := ParsePrivateKeyPEMWithPassword([]byte(encryptedPEM), []byte("pony"))
	require.Error(t, err)

	_, err = ParsePrivateKeyPEMWithPassword([]byte(encryptedPEM), []byte("ponies"))
	require.NoError(t, err)

	_, err = ParsePrivateKeyPEMWithPassword([]byte(decryptedPEM), nil)
	require.NoError(t, err)
}

func TestConvertECPrivateKeyPEM(t *testing.T) {
	_, err := ConvertECPrivateKeyPEM([]byte(`garbage pem`))
	require.Error(t, err)

	_, err = ConvertECPrivateKeyPEM([]byte(`-----BEGIN EC PRIVATE KEY-----
garbage key
-----END EC PRIVATE KEY-----`))
	require.Error(t, err)

	out, err := ConvertECPrivateKeyPEM([]byte(ecKeyPEM))
	require.NoError(t, err)

	_, err = helpers.ParsePrivateKeyPEM([]byte(ecKeyPEM))
	require.NoError(t, err)
	_, err = helpers.ParsePrivateKeyPEM(out)
	require.NoError(t, err)
	require.Equal(t, []byte(decryptedPEM), out)
}

func TestConvertToECPrivateKeyPEM(t *testing.T) {
	_, err := ConvertToECPrivateKeyPEM([]byte(`garbage pem`))
	require.Error(t, err)

	_, err = ConvertToECPrivateKeyPEM([]byte(`-----BEGIN PRIVATE KEY-----
garbage key
-----END PRIVATE KEY-----`))
	require.Error(t, err)

	out, err := ConvertToECPrivateKeyPEM([]byte(decryptedPEM))
	require.NoError(t, err)

	_, err = helpers.ParsePrivateKeyPEM([]byte(decryptedPEM))
	require.NoError(t, err)
	_, err = helpers.ParsePrivateKeyPEM(out)
	require.NoError(t, err)
	require.Equal(t, []byte(ecKeyPEM), out)
}
