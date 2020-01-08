package ca_test

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/keyutils"
	"github.com/docker/swarmkit/ca/pkcs8"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/stretchr/testify/require"
)

// can read and write tls keys that aren't encrypted, and that are encrypted.  without
// a pem header manager, the headers are all preserved and not overwritten
func TestKeyReadWriter(t *testing.T) {
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	expectedKey := key

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir, "subdir")) // to make sure subdirectories are created

	checkCanReadWithKEK := func(kek []byte) *ca.KeyReadWriter {
		k := ca.NewKeyReadWriter(path.Node, kek, nil)
		readCert, readKey, err := k.Read()
		require.NoError(t, err)
		require.Equal(t, cert, readCert)
		require.Equal(t, expectedKey, readKey, "Expected %s, Got %s", string(expectedKey), string(readKey))
		return k
	}

	k := ca.NewKeyReadWriter(path.Node, nil, nil)

	// can't read things that don't exist
	_, _, err = k.Read()
	require.Error(t, err)

	// can write an unencrypted key with no updates
	require.NoError(t, k.Write(cert, expectedKey, nil))

	// can read unencrypted
	k = checkCanReadWithKEK(nil)
	_, kekData := k.GetCurrentState()
	require.EqualValues(t, 0, kekData.Version) // the first version was 0

	// write a key with headers to the key to make sure they're cleaned
	keyBlock, _ := pem.Decode(expectedKey)
	require.NotNil(t, keyBlock)
	keyBlock.Headers = map[string]string{"hello": "world"}
	expectedKey = pem.EncodeToMemory(keyBlock)
	// write a version, but that's not what we'd expect back once we read
	keyBlock.Headers["kek-version"] = "8"
	require.NoError(t, ioutil.WriteFile(path.Node.Key, pem.EncodeToMemory(keyBlock), 0600))

	// if a kek is provided, we can still read unencrypted keys, and read
	// the provided version
	k = checkCanReadWithKEK([]byte("original kek"))
	_, kekData = k.GetCurrentState()
	require.EqualValues(t, 8, kekData.Version)

	// we can update the kek and write at the same time
	require.NoError(t, k.Write(cert, key, &ca.KEKData{KEK: []byte("new kek!"), Version: 3}))

	// the same kek can still read, and will continue to write with this key if
	// no further kek updates are provided
	_, _, err = k.Read()
	require.NoError(t, err)
	require.NoError(t, k.Write(cert, expectedKey, nil))

	expectedKey = key

	// without the right kek, we can't read
	k = ca.NewKeyReadWriter(path.Node, []byte("original kek"), nil)
	_, _, err = k.Read()
	require.Error(t, err)

	// same new key, just for sanity
	k = checkCanReadWithKEK([]byte("new kek!"))
	_, kekData = k.GetCurrentState()
	require.EqualValues(t, 3, kekData.Version)

	// we can also change the kek back to nil, which means the key is unencrypted
	require.NoError(t, k.Write(cert, key, &ca.KEKData{KEK: nil}))
	k = checkCanReadWithKEK(nil)
	_, kekData = k.GetCurrentState()
	require.EqualValues(t, 0, kekData.Version)
}

type testHeaders struct {
	setHeaders func(map[string]string, ca.KEKData) (ca.PEMKeyHeaders, error)
	newHeaders func(ca.KEKData) (map[string]string, error)
}

func (p testHeaders) UnmarshalHeaders(h map[string]string, k ca.KEKData) (ca.PEMKeyHeaders, error) {
	if p.setHeaders != nil {
		return p.setHeaders(h, k)
	}
	return nil, fmt.Errorf("set header error")
}

func (p testHeaders) MarshalHeaders(k ca.KEKData) (map[string]string, error) {
	if p.newHeaders != nil {
		return p.newHeaders(k)
	}
	return nil, fmt.Errorf("update header error")
}

func (p testHeaders) UpdateKEK(ca.KEKData, ca.KEKData) ca.PEMKeyHeaders {
	return p
}

// KeyReaderWriter makes a call to a get headers updater, if write is called,
// and set headers, if read is called.  The KEK version header is always preserved
// no matter what.
func TestKeyReadWriterWithPemHeaderManager(t *testing.T) {
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	// write a key with headers to the key to make sure it gets overwritten
	keyBlock, _ := pem.Decode(key)
	require.NotNil(t, keyBlock)
	keyBlock.Headers = map[string]string{"hello": "world"}
	key = pem.EncodeToMemory(keyBlock)

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir, "subdir")) // to make sure subdirectories are created

	// if if getting new headers fail, writing a key fails, and the key does not rotate
	var count int
	badKEKData := ca.KEKData{KEK: []byte("failed kek"), Version: 3}
	k := ca.NewKeyReadWriter(path.Node, nil, testHeaders{newHeaders: func(k ca.KEKData) (map[string]string, error) {
		if count == 0 {
			count++
			require.Equal(t, badKEKData, k)
			return nil, fmt.Errorf("fail")
		}
		require.Equal(t, ca.KEKData{}, k)
		return nil, nil
	}})
	// first write will fail
	require.Error(t, k.Write(cert, key, &badKEKData))
	// the stored kek data will be not be updated because the write failed
	_, kekData := k.GetCurrentState()
	require.Equal(t, ca.KEKData{}, kekData)
	// second write will succeed, using the original kek (nil)
	require.NoError(t, k.Write(cert, key, nil))

	var (
		headers map[string]string
		kek     ca.KEKData
	)

	// if setting headers fail, reading fails
	k = ca.NewKeyReadWriter(path.Node, nil, testHeaders{setHeaders: func(map[string]string, ca.KEKData) (ca.PEMKeyHeaders, error) {
		return nil, fmt.Errorf("nope")
	}})
	_, _, err = k.Read()
	require.Error(t, err)

	k = ca.NewKeyReadWriter(path.Node, nil, testHeaders{setHeaders: func(h map[string]string, k ca.KEKData) (ca.PEMKeyHeaders, error) {
		headers = h
		kek = k
		return testHeaders{}, nil
	}})

	_, _, err = k.Read()
	require.NoError(t, err)
	require.Equal(t, ca.KEKData{}, kek)
	require.Equal(t, keyBlock.Headers, headers)

	// writing new headers is called with existing headers, and will write a key that has the headers
	// returned by the header update function
	k = ca.NewKeyReadWriter(path.Node, []byte("oldKek"), testHeaders{newHeaders: func(kek ca.KEKData) (map[string]string, error) {
		require.Equal(t, []byte("newKEK"), kek.KEK)
		return map[string]string{"updated": "headers"}, nil
	}})
	require.NoError(t, k.Write(cert, key, &ca.KEKData{KEK: []byte("newKEK"), Version: 2}))

	// make sure headers were correctly set
	k = ca.NewKeyReadWriter(path.Node, []byte("newKEK"), testHeaders{setHeaders: func(h map[string]string, k ca.KEKData) (ca.PEMKeyHeaders, error) {
		headers = h
		kek = k
		return testHeaders{}, nil
	}})
	_, _, err = k.Read()
	require.NoError(t, err)
	require.Equal(t, ca.KEKData{KEK: []byte("newKEK"), Version: 2}, kek)

	_, kekData = k.GetCurrentState()
	require.Equal(t, kek, kekData)
	require.Equal(t, map[string]string{"updated": "headers"}, headers)
}

func TestKeyReadWriterViewAndUpdateHeaders(t *testing.T) {
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir))

	// write a key with headers to the key to make sure it gets passed when reading/writing headers
	keyBlock, _ := pem.Decode(key)
	require.NotNil(t, keyBlock)
	keyBlock.Headers = map[string]string{"hello": "world"}
	key = pem.EncodeToMemory(keyBlock)
	require.NoError(t, ioutil.WriteFile(path.Node.Cert, cert, 0644))
	require.NoError(t, ioutil.WriteFile(path.Node.Key, key, 0600))

	// if the update headers callback function fails, updating headers fails
	k := ca.NewKeyReadWriter(path.Node, nil, nil)
	err = k.ViewAndUpdateHeaders(func(h ca.PEMKeyHeaders) (ca.PEMKeyHeaders, error) {
		require.Nil(t, h)
		return nil, fmt.Errorf("nope")
	})
	require.Error(t, err)
	require.Equal(t, "nope", err.Error())

	// updating headers succeed and is called with the latest kek data
	err = k.ViewAndUpdateHeaders(func(h ca.PEMKeyHeaders) (ca.PEMKeyHeaders, error) {
		require.Nil(t, h)
		return testHeaders{newHeaders: func(kek ca.KEKData) (map[string]string, error) {
			return map[string]string{"updated": "headers"}, nil
		}}, nil
	})
	require.NoError(t, err)

	k = ca.NewKeyReadWriter(path.Node, nil, testHeaders{setHeaders: func(h map[string]string, k ca.KEKData) (ca.PEMKeyHeaders, error) {
		require.Equal(t, map[string]string{"updated": "headers"}, h)
		require.Equal(t, ca.KEKData{}, k)
		return testHeaders{}, nil
	}})
	_, _, err = k.Read()
	require.NoError(t, err)

	// we can also update headers on an encrypted key
	k = ca.NewKeyReadWriter(path.Node, []byte("kek"), nil)
	require.NoError(t, k.Write(cert, key, nil))

	err = k.ViewAndUpdateHeaders(func(h ca.PEMKeyHeaders) (ca.PEMKeyHeaders, error) {
		require.Nil(t, h)
		return testHeaders{newHeaders: func(kek ca.KEKData) (map[string]string, error) {
			require.Equal(t, ca.KEKData{KEK: []byte("kek")}, kek)
			return map[string]string{"updated": "headers"}, nil
		}}, nil
	})
	require.NoError(t, err)

	k = ca.NewKeyReadWriter(path.Node, []byte("kek"), testHeaders{setHeaders: func(h map[string]string, k ca.KEKData) (ca.PEMKeyHeaders, error) {
		require.Equal(t, map[string]string{"updated": "headers"}, h)
		require.Equal(t, ca.KEKData{KEK: []byte("kek")}, k)
		return testHeaders{}, nil
	}})
	_, _, err = k.Read()
	require.NoError(t, err)
}

func TestKeyReadWriterViewAndRotateKEK(t *testing.T) {
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir))

	// write a key with headers to the key to make sure it gets passed when reading/writing headers
	keyBlock, _ := pem.Decode(key)
	require.NotNil(t, keyBlock)
	keyBlock.Headers = map[string]string{"hello": "world"}
	key = pem.EncodeToMemory(keyBlock)
	require.NoError(t, ca.NewKeyReadWriter(path.Node, nil, nil).Write(cert, key, nil))

	// if if getting new kek and headers fail, rotating a KEK fails, and the kek does not rotate
	k := ca.NewKeyReadWriter(path.Node, nil, nil)
	require.Error(t, k.ViewAndRotateKEK(func(k ca.KEKData, h ca.PEMKeyHeaders) (ca.KEKData, ca.PEMKeyHeaders, error) {
		require.Equal(t, ca.KEKData{}, k)
		require.Nil(t, h)
		return ca.KEKData{}, nil, fmt.Errorf("Nope")
	}))

	// writing new headers will write a key that has the headers returned by the header update function
	k = ca.NewKeyReadWriter(path.Node, []byte("oldKEK"), nil)
	require.NoError(t, k.ViewAndRotateKEK(func(k ca.KEKData, h ca.PEMKeyHeaders) (ca.KEKData, ca.PEMKeyHeaders, error) {
		require.Equal(t, ca.KEKData{KEK: []byte("oldKEK")}, k)
		require.Nil(t, h)
		return ca.KEKData{KEK: []byte("newKEK"), Version: uint64(2)},
			testHeaders{newHeaders: func(kek ca.KEKData) (map[string]string, error) {
				require.Equal(t, []byte("newKEK"), kek.KEK)
				return map[string]string{"updated": "headers"}, nil
			}}, nil
	}))

	// ensure the key has been re-encrypted and we can read it
	k = ca.NewKeyReadWriter(path.Node, nil, nil)
	_, _, err = k.Read()
	require.Error(t, err)

	var headers map[string]string

	k = ca.NewKeyReadWriter(path.Node, []byte("newKEK"), testHeaders{setHeaders: func(h map[string]string, _ ca.KEKData) (ca.PEMKeyHeaders, error) {
		headers = h
		return testHeaders{}, nil
	}})
	_, _, err = k.Read()
	require.NoError(t, err)
	require.Equal(t, map[string]string{"updated": "headers"}, headers)
}

// If we abort in the middle of writing the key and cert, such that only the key is written
// to the final location, when we read we can still read the cert from the temporary
// location.
func TestTwoPhaseReadWrite(t *testing.T) {
	cert1, _, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	cert2, key2, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir))
	krw := ca.NewKeyReadWriter(path.Node, nil, nil)

	// put a directory in the location where the cert goes, so we can't actually move
	// the cert from the temporary location to the final location.
	require.NoError(t, os.Mkdir(filepath.Join(path.Node.Cert), 0755))
	require.Error(t, krw.Write(cert2, key2, nil))

	// the temp cert file should exist
	tempCertPath := filepath.Join(filepath.Dir(path.Node.Cert), "."+filepath.Base(path.Node.Cert))
	readCert, err := ioutil.ReadFile(tempCertPath)
	require.NoError(t, err)
	require.Equal(t, cert2, readCert)

	// remove the directory, to simulate it failing to write the first time
	os.RemoveAll(path.Node.Cert)
	readCert, readKey, err := krw.Read()
	require.NoError(t, err)
	require.Equal(t, cert2, readCert)
	require.Equal(t, key2, readKey)
	// the cert should have been moved to its proper location
	_, err = os.Stat(tempCertPath)
	require.True(t, os.IsNotExist(err))

	// If the cert in the proper location doesn't match the key, the temp location is checked
	require.NoError(t, ioutil.WriteFile(tempCertPath, cert2, 0644))
	require.NoError(t, ioutil.WriteFile(path.Node.Cert, cert1, 0644))
	readCert, readKey, err = krw.Read()
	require.NoError(t, err)
	require.Equal(t, cert2, readCert)
	require.Equal(t, key2, readKey)
	// the cert should have been moved to its proper location
	_, err = os.Stat(tempCertPath)
	require.True(t, os.IsNotExist(err))

	// If the cert in the temp location also doesn't match, the failure matching the
	// correctly-located cert is returned
	require.NoError(t, os.Remove(path.Node.Cert))
	require.NoError(t, ioutil.WriteFile(tempCertPath, cert1, 0644)) // mismatching cert
	_, _, err = krw.Read()
	require.True(t, os.IsNotExist(err))
	// the cert should have been removed
	_, err = os.Stat(tempCertPath)
	require.True(t, os.IsNotExist(err))
}

func TestKeyReadWriterMigrate(t *testing.T) {
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir))

	// if the key exists in an old location, migrate it from there.
	tempKeyPath := filepath.Join(filepath.Dir(path.Node.Key), "."+filepath.Base(path.Node.Key))
	require.NoError(t, ioutil.WriteFile(path.Node.Cert, cert, 0644))
	require.NoError(t, ioutil.WriteFile(tempKeyPath, key, 0600))

	krw := ca.NewKeyReadWriter(path.Node, nil, nil)
	require.NoError(t, krw.Migrate())
	_, err = os.Stat(tempKeyPath)
	require.True(t, os.IsNotExist(err)) // it's been moved to the right place
	_, _, err = krw.Read()
	require.NoError(t, err)

	// migrate does not affect any existing files
	dirList, err := ioutil.ReadDir(filepath.Dir(path.Node.Key))
	require.NoError(t, err)
	require.NoError(t, krw.Migrate())
	dirList2, err := ioutil.ReadDir(filepath.Dir(path.Node.Key))
	require.NoError(t, err)
	require.Equal(t, dirList, dirList2)
	_, _, err = krw.Read()
	require.NoError(t, err)
}

type downgradeTestCase struct {
	encrypted bool
	pkcs8     bool
	errorStr  string
}

func testKeyReadWriterDowngradeKeyCase(t *testing.T, tc downgradeTestCase) error {
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	if !tc.pkcs8 {
		key, err = pkcs8.ConvertToECPrivateKeyPEM(key)
		require.NoError(t, err)
	}

	var kek []byte
	if tc.encrypted {
		block, _ := pem.Decode(key)
		require.NotNil(t, block)

		kek = []byte("kek")
		block, err = keyutils.Default.EncryptPEMBlock(block.Bytes, kek)
		require.NoError(t, err)

		key = pem.EncodeToMemory(block)
	}

	tempdir, err := ioutil.TempDir("", "KeyReadWriterDowngrade")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir))

	block, _ := pem.Decode(key)
	require.NotNil(t, block)

	// add kek-version to later check if it is still there
	block.Headers["kek-version"] = "5"

	key = pem.EncodeToMemory(block)
	require.NoError(t, ioutil.WriteFile(path.Node.Cert, cert, 0644))
	require.NoError(t, ioutil.WriteFile(path.Node.Key, key, 0600))

	// if the update headers callback function fails, updating headers fails
	k := ca.NewKeyReadWriter(path.Node, kek, nil)
	if err := k.DowngradeKey(); err != nil {
		return err
	}

	// read the key directly from fs so we can check if key
	key, err = ioutil.ReadFile(path.Node.Key)
	require.NoError(t, err)

	keyBlock, _ := pem.Decode(key)
	require.NotNil(t, block)
	require.False(t, keyutils.IsPKCS8(keyBlock.Bytes))

	if tc.encrypted {
		require.True(t, keyutils.IsEncryptedPEMBlock(keyBlock))
	}
	require.Equal(t, "5", keyBlock.Headers["kek-version"])

	// check if KeyReaderWriter can read the key
	_, _, err = k.Read()
	require.NoError(t, err)
	return nil
}

func TestKeyReadWriterDowngradeKey(t *testing.T) {
	invalid := []downgradeTestCase{
		{
			encrypted: false,
			pkcs8:     false,
			errorStr:  "key is already downgraded to PKCS#1",
		}, {
			encrypted: true,
			pkcs8:     false,
			errorStr:  "key is already downgraded to PKCS#1",
		},
	}

	for _, c := range invalid {
		err := testKeyReadWriterDowngradeKeyCase(t, c)
		require.Error(t, err)
		require.EqualError(t, err, c.errorStr)
	}

	valid := []downgradeTestCase{
		{
			encrypted: false,
			pkcs8:     true,
		}, {
			encrypted: true,
			pkcs8:     true,
		},
	}

	for _, c := range valid {
		err := testKeyReadWriterDowngradeKeyCase(t, c)
		require.NoError(t, err)
	}
}

// In FIPS mode, when reading a PKCS1 encrypted key, a PKCS1 error is returned as opposed
// to any other type of invalid KEK error
func TestKeyReadWriterReadNonFIPS(t *testing.T) {
	t.Parallel()
	cert, key, err := testutils.CreateRootCertAndKey("cn")
	require.NoError(t, err)

	key, err = pkcs8.ConvertToECPrivateKeyPEM(key)
	require.NoError(t, err)

	tempdir, err := ioutil.TempDir("", "KeyReadWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	path := ca.NewConfigPaths(filepath.Join(tempdir, "subdir")) // to make sure subdirectories are created

	k := ca.NewKeyReadWriter(path.Node, nil, nil)
	k.SetKeyFormatter(keyutils.FIPS)

	// can write an unencrypted PKCS1 key with no issues
	require.NoError(t, k.Write(cert, key, nil))
	// can read the unencrypted key with no issues
	readCert, readKey, err := k.Read()
	require.NoError(t, err)
	require.Equal(t, cert, readCert)
	require.Equal(t, key, readKey)

	// cannot write an encrypted PKCS1 key
	passphrase := []byte("passphrase")
	require.Equal(t, keyutils.ErrFIPSUnsupportedKeyFormat, k.Write(cert, key, &ca.KEKData{KEK: passphrase}))

	k.SetKeyFormatter(keyutils.Default)
	require.NoError(t, k.Write(cert, key, &ca.KEKData{KEK: passphrase}))

	// cannot read an encrypted PKCS1 key
	k.SetKeyFormatter(keyutils.FIPS)
	_, _, err = k.Read()
	require.Equal(t, keyutils.ErrFIPSUnsupportedKeyFormat, err)

	k.SetKeyFormatter(keyutils.Default)
	_, _, err = k.Read()
	require.NoError(t, err)
}
