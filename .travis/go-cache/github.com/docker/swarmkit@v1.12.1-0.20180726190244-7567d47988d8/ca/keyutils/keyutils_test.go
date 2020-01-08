package keyutils

import (
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	decryptedPKCS1 = `-----BEGIN EC PRIVATE KEY-----
MIHbAgEBBEHECF7HdJ4QZ7Dx0FBzzV/6vgI+bZNZGWtmbVwPIMu/bZE1p2qz5HGS
EFsmor5X6t7KYLa4nQNqbloWaneRNNukk6AHBgUrgQQAI6GBiQOBhgAEAW4hBUpI
+ckv40lP6HIUTr/71yhrZWjCWGh84xNk8LxNA54oy4DV4hS7E9+NLHKJrwnLDlnG
FR9il6zgU/9IsJdWAVcqVY7vsOKs8dquQ1HLXcOos22TOXbQne3Ua66HC0mjJ9Xp
LrnqZrqoHphZCknCX9HFSrlvdq6PEBSaCgfe3dd/
-----END EC PRIVATE KEY-----
`
	encryptedPKCS1 = `-----BEGIN EC PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,8EE2B3B5A92822309E6157EBFFB238ED

clpdzQaCjXy2ZNLEsiGSpt0//DRdO1haJ4wrDTrhb78npiWrWjVsyAEwBoSPRwPW
ZnGKjAV+tv7w4XujycwijsSBVCzGvCbMYnzO+n0zApD6eo1SF/bRCZqEPcWDnsCK
UtLuqa3o8F0q3Bh8woOJ6NOq8dNWA2XHNkNhs77aqTh+bDR+jruDjFDB5/HZxDU2
aCpI96TeakB+8upn+/1wkpxfAJLpbkOdWDIgTEMhhwZUBQocoZezEORn4JIpYknY
0fOJaoM+gMMVLDPvXWUZFulP+2TpIOsHWspY2D4mYUE=
-----END EC PRIVATE KEY-----
`
	decryptedPKCS8 = `-----BEGIN PRIVATE KEY-----
MHgCAQAwEAYHKoZIzj0CAQYFK4EEACEEYTBfAgEBBBwCTYvOWrsYitgVHwD6F4GH
1re5Oe05CtZ4PUgkoTwDOgAETRlz5X662R8MX3tcoTTZiE2psZScMQNo6X/6gH+L
5xPO1GTcpbAt8U+ULn/4S5Bgq+WIgA8bI4g=
-----END PRIVATE KEY-----
`
	encryptedPKCS8 = `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHOMEkGCSqGSIb3DQEFDTA8MBsGCSqGSIb3DQEFDDAOBAiGRncJ5A+72AICCAAw
HQYJYIZIAWUDBAEqBBA0iGGDrKda4SbsQlW8hgiOBIGA1rDEtNqghfQ+8AtdB7kY
US05ElIO2ooXviNo0M36Shltv+1ntd/Qxn+El1B+0BT8MngB8yBV6oFach1dfKvR
PkeX/+bOnd1WTKMx3IPNMWxbA9YPTeoaObaKI7awvI03o51HLd+a5BuHJ55N2CX4
aMbljbOLAjpZS3/VnQteab4=
-----END ENCRYPTED PRIVATE KEY-----
`
	decryptedPKCS8Block, _ = pem.Decode([]byte(decryptedPKCS8))
	encryptedPKCS8Block, _ = pem.Decode([]byte(encryptedPKCS8))
	decryptedPKCS1Block, _ = pem.Decode([]byte(decryptedPKCS1))
	encryptedPKCS1Block, _ = pem.Decode([]byte(encryptedPKCS1))
)

func TestIsPKCS8(t *testing.T) {
	// Check PKCS8 keys
	assert.True(t, IsPKCS8([]byte(decryptedPKCS8Block.Bytes)))
	assert.True(t, IsPKCS8([]byte(encryptedPKCS8Block.Bytes)))

	// Check PKCS1 keys
	assert.False(t, IsPKCS8([]byte(decryptedPKCS1Block.Bytes)))
	assert.False(t, IsPKCS8([]byte(encryptedPKCS1Block.Bytes)))
}

func TestIsEncryptedPEMBlock(t *testing.T) {
	// Check PKCS8
	assert.False(t, IsEncryptedPEMBlock(decryptedPKCS8Block))
	assert.True(t, IsEncryptedPEMBlock(encryptedPKCS8Block))

	// Check PKCS1
	assert.False(t, IsEncryptedPEMBlock(decryptedPKCS1Block))
	assert.True(t, IsEncryptedPEMBlock(encryptedPKCS1Block))
}

func TestDecryptPEMBlock(t *testing.T) {
	// Check PKCS8 keys in both FIPS and non-FIPS mode
	for _, util := range []Formatter{Default, FIPS} {
		_, err := util.DecryptPEMBlock(encryptedPKCS8Block, []byte("pony"))
		require.Error(t, err)

		decryptedDer, err := util.DecryptPEMBlock(encryptedPKCS8Block, []byte("ponies"))
		require.NoError(t, err)
		require.Equal(t, decryptedPKCS8Block.Bytes, decryptedDer)
	}

	// Check PKCS1 keys in non-FIPS mode
	_, err := Default.DecryptPEMBlock(encryptedPKCS1Block, []byte("pony"))
	require.Error(t, err)

	decryptedDer, err := Default.DecryptPEMBlock(encryptedPKCS1Block, []byte("ponies"))
	require.NoError(t, err)
	require.Equal(t, decryptedPKCS1Block.Bytes, decryptedDer)

	// Try to decrypt PKCS1 in FIPS
	_, err = FIPS.DecryptPEMBlock(encryptedPKCS1Block, []byte("ponies"))
	require.Error(t, err)
}

func TestEncryptPEMBlock(t *testing.T) {
	// Check PKCS8 keys in both FIPS and non-FIPS mode
	for _, util := range []Formatter{Default, FIPS} {
		encryptedBlock, err := util.EncryptPEMBlock(decryptedPKCS8Block.Bytes, []byte("knock knock"))
		require.NoError(t, err)

		// Try to decrypt the same encrypted block
		_, err = util.DecryptPEMBlock(encryptedBlock, []byte("hey there"))
		require.Error(t, err)

		decryptedDer, err := Default.DecryptPEMBlock(encryptedBlock, []byte("knock knock"))
		require.NoError(t, err)
		require.Equal(t, decryptedPKCS8Block.Bytes, decryptedDer)
	}

	// Check PKCS1 keys in non FIPS mode
	encryptedBlock, err := Default.EncryptPEMBlock(decryptedPKCS1Block.Bytes, []byte("knock knock"))
	require.NoError(t, err)

	// Try to decrypt the same encrypted block
	_, err = Default.DecryptPEMBlock(encryptedBlock, []byte("hey there"))
	require.Error(t, err)

	decryptedDer, err := Default.DecryptPEMBlock(encryptedBlock, []byte("knock knock"))
	require.NoError(t, err)
	require.Equal(t, decryptedPKCS1Block.Bytes, decryptedDer)

	// Try to encrypt PKCS1
	_, err = FIPS.EncryptPEMBlock(decryptedPKCS1Block.Bytes, []byte("knock knock"))
	require.Error(t, err)
}

func TestParsePrivateKeyPEMWithPassword(t *testing.T) {
	// Check PKCS8 keys in both FIPS and non-FIPS mode
	for _, util := range []Formatter{Default, FIPS} {
		_, err := util.ParsePrivateKeyPEMWithPassword([]byte(encryptedPKCS8), []byte("pony"))
		require.Error(t, err)

		_, err = util.ParsePrivateKeyPEMWithPassword([]byte(encryptedPKCS8), []byte("ponies"))
		require.NoError(t, err)

		_, err = util.ParsePrivateKeyPEMWithPassword([]byte(decryptedPKCS8), nil)
		require.NoError(t, err)
	}

	// Check PKCS1 keys in non-FIPS mode
	_, err := Default.ParsePrivateKeyPEMWithPassword([]byte(encryptedPKCS1), []byte("pony"))
	require.Error(t, err)

	_, err = Default.ParsePrivateKeyPEMWithPassword([]byte(encryptedPKCS1), []byte("ponies"))
	require.NoError(t, err)

	_, err = Default.ParsePrivateKeyPEMWithPassword([]byte(decryptedPKCS1), nil)
	require.NoError(t, err)

	// Try to parse PKCS1 in FIPS mode
	_, err = FIPS.ParsePrivateKeyPEMWithPassword([]byte(encryptedPKCS1), []byte("ponies"))
	require.Error(t, err)
}
