package controlapi

import (
	"context"
	"crypto/x509"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"encoding/pem"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type rootCARotationTestCase struct {
	rootCA   api.RootCA
	caConfig api.CAConfig

	// what to expect if the validate and update succeeds - we can't always check that everything matches, for instance if
	// random values for join tokens or cross signed certs, or generated root rotation cert/key,
	// are expected
	expectRootCA                api.RootCA
	expectJoinTokenChange       bool
	expectGeneratedRootRotation bool
	expectGeneratedCross        bool
	description                 string // in case an expectation fails

	// what error string to expect if the validate fails
	expectErrorString string
}

var initialLocalRootCA = api.RootCA{
	CACert:     testutils.ECDSA256SHA256Cert,
	CAKey:      testutils.ECDSA256Key,
	CACertHash: "DEADBEEF",
	JoinTokens: api.JoinTokens{
		Worker:  "SWMTKN-1-worker",
		Manager: "SWMTKN-1-manager",
	},
}
var rotationCert, rotationKey = testutils.ECDSACertChain[2], testutils.ECDSACertChainKeys[2]

func uglifyOnePEM(pemBytes []byte) []byte {
	pemBlock, _ := pem.Decode(pemBytes)
	pemBlock.Headers = map[string]string{
		"this": "should",
		"be":   "removed",
	}
	return append(append([]byte("\n\t   "), pem.EncodeToMemory(pemBlock)...), []byte("   \t")...)
}

func getSecurityConfig(t *testing.T, localRootCA *ca.RootCA, cluster *api.Cluster) *ca.SecurityConfig {
	tempdir, err := ioutil.TempDir("", "test-validate-CA")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := ca.NewConfigPaths(tempdir)
	secConfig, cancel, err := localRootCA.CreateSecurityConfig(context.Background(), ca.NewKeyReadWriter(paths.Node, nil, nil), ca.CertificateRequestConfig{})
	require.NoError(t, err)
	cancel()
	return secConfig
}

func TestValidateCAConfigInvalidValues(t *testing.T) {
	t.Parallel()
	localRootCA, err := ca.NewRootCA(initialLocalRootCA.CACert, initialLocalRootCA.CACert, initialLocalRootCA.CAKey,
		ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	initialExternalRootCA := initialLocalRootCA
	initialExternalRootCA.CAKey = nil

	crossSigned, err := localRootCA.CrossSignCACertificate(rotationCert)
	require.NoError(t, err)

	initExternalRootCAWithRotation := initialExternalRootCA
	initExternalRootCAWithRotation.RootRotation = &api.RootRotation{
		CACert:            rotationCert,
		CAKey:             rotationKey,
		CrossSignedCACert: crossSigned,
	}

	initWithExternalRootRotation := initialLocalRootCA
	initWithExternalRootRotation.RootRotation = &api.RootRotation{
		CACert:            rotationCert,
		CrossSignedCACert: crossSigned,
	}

	// set up 2 external CAs that can be contacted for signing
	tempdir, err := ioutil.TempDir("", "test-validate-CA")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	initExtServer, err := testutils.NewExternalSigningServer(localRootCA, tempdir)
	require.NoError(t, err)
	defer initExtServer.Stop()

	// we need to accept client certs from the original cert
	rotationRootCA, err := ca.NewRootCA(append(initialLocalRootCA.CACert, rotationCert...), rotationCert, rotationKey,
		ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	rotateExtServer, err := testutils.NewExternalSigningServer(rotationRootCA, tempdir)
	require.NoError(t, err)
	defer rotateExtServer.Stop()

	for _, invalid := range []rootCARotationTestCase{
		{
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCAKey: initialLocalRootCA.CAKey,
			},
			expectErrorString: "the signing CA cert must also be provided",
		},
		{
			rootCA: initExternalRootCAWithRotation, // even if a root rotation is already in progress, the current CA external URL must be present
			caConfig: api.CAConfig{
				ExternalCAs: []*api.ExternalCA{
					{
						URL:      initExtServer.URL,
						CACert:   initialLocalRootCA.CACert,
						Protocol: 3, // wrong protocol
					},
					{
						URL:    initExtServer.URL,
						CACert: rotationCert, // wrong cert
					},
				},
			},
			expectErrorString: "there must be at least one valid, reachable external CA corresponding to the current CA certificate",
		},
		{
			rootCA: initialExternalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: rotationCert, // even if there's a desired cert, the current CA external URL must be present
				ExternalCAs: []*api.ExternalCA{ // right certs, but invalid URLs in several ways
					{
						URL:    rotateExtServer.URL,
						CACert: initialExternalRootCA.CACert,
					},
					{
						URL:    "invalidurl",
						CACert: initialExternalRootCA.CACert,
					},
					{
						URL:    "https://too:many:colons:1:2:3",
						CACert: initialExternalRootCA.CACert,
					},
				},
			},
			expectErrorString: "there must be at least one valid, reachable external CA corresponding to the current CA certificate",
		},
		{
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: rotationCert,
				ExternalCAs: []*api.ExternalCA{
					{
						URL:      rotateExtServer.URL,
						CACert:   rotationCert,
						Protocol: 3, // wrong protocol
					},
					{
						URL: rotateExtServer.URL,
						// wrong cert because no cert is assumed to be the current root CA cert
					},
				},
			},
			expectErrorString: "there must be at least one valid, reachable external CA corresponding to the desired CA certificate",
		},
		{
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: rotationCert,
				ExternalCAs: []*api.ExternalCA{ // right certs, but invalid URLs in several ways
					{
						URL:    initExtServer.URL,
						CACert: rotationCert,
					},
					{
						URL:    "invalidurl",
						CACert: rotationCert,
					},
					{
						URL:    "https://too:many:colons:1:2:3",
						CACert: initialExternalRootCA.CACert,
					},
				},
			},
			expectErrorString: "there must be at least one valid, reachable external CA corresponding to the desired CA certificate",
		},
		{
			rootCA: initWithExternalRootRotation,
			caConfig: api.CAConfig{ // no forceRotate change, no explicit signing cert change
				ExternalCAs: []*api.ExternalCA{
					{
						URL:      rotateExtServer.URL,
						CACert:   rotationCert,
						Protocol: 3, // wrong protocol
					},
					{
						URL:    rotateExtServer.URL,
						CACert: initialLocalRootCA.CACert, // wrong cert
					},
				},
			},
			expectErrorString: "there must be at least one valid, reachable external CA corresponding to the next CA certificate",
		},
		{
			rootCA: initWithExternalRootRotation,
			caConfig: api.CAConfig{ // no forceRotate change, no explicit signing cert change
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    initExtServer.URL,
						CACert: rotationCert,
						// right CA cert, but the server cert is not signed by this CA cert
					},
					{
						URL:    "invalidurl",
						CACert: rotationCert,
						// right CA cert, but invalid URL
					},
				},
			},
			expectErrorString: "there must be at least one valid, reachable external CA corresponding to the next CA certificate",
		},
		{
			rootCA: initialExternalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: rotationCert,
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    initExtServer.URL,
						CACert: initialLocalRootCA.CACert, // current cert
					},
					{
						URL:    rotateExtServer.URL,
						CACert: rotationCert, //new cert
					},
				},
			},
			expectErrorString: "rotating from one external CA to a different external CA is not supported",
		},
		{
			rootCA: initialExternalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: rotationCert,
				ExternalCAs: []*api.ExternalCA{
					{
						URL: initExtServer.URL,
						// no cert means the current cert
					},
					{
						URL:    rotateExtServer.URL,
						CACert: rotationCert, //new cert
					},
				},
			},
			expectErrorString: "rotating from one external CA to a different external CA is not supported",
		},
		{
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: append(rotationCert, initialLocalRootCA.CACert...),
				SigningCAKey:  rotationKey,
			},
			expectErrorString: "cannot contain multiple certificates",
		},
		{
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: testutils.ReDateCert(t, rotationCert, rotationCert, rotationKey,
					time.Now().Add(-1*time.Minute), time.Now().Add(364*helpers.OneDay)),
				SigningCAKey: rotationKey,
			},
			expectErrorString: "expires too soon",
		},
		{
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: initialLocalRootCA.CACert,
				SigningCAKey:  testutils.ExpiredKey, // same cert but mismatching key
			},
			expectErrorString: "certificate key mismatch",
		},
		{
			// this is just one class of failures caught by NewRootCA, not going to bother testing others, since they are
			// extensively tested in NewRootCA
			rootCA: initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: testutils.ExpiredCert,
				SigningCAKey:  testutils.ExpiredKey,
			},
			expectErrorString: "expired",
		},
	} {
		cluster := &api.Cluster{
			RootCA: invalid.rootCA,
			Spec: api.ClusterSpec{
				CAConfig: invalid.caConfig,
			},
		}
		secConfig := getSecurityConfig(t, &localRootCA, cluster)
		_, err := validateCAConfig(context.Background(), secConfig, cluster)
		require.Error(t, err, invalid.expectErrorString)
		require.Equal(t, codes.InvalidArgument, grpc.Code(err), invalid.expectErrorString)
		require.Contains(t, grpc.ErrorDesc(err), invalid.expectErrorString)
	}
}

func runValidTestCases(t *testing.T, testcases []*rootCARotationTestCase, localRootCA *ca.RootCA) {
	for _, valid := range testcases {
		cluster := &api.Cluster{
			RootCA: *valid.rootCA.Copy(),
			Spec: api.ClusterSpec{
				CAConfig: valid.caConfig,
			},
		}
		secConfig := getSecurityConfig(t, localRootCA, cluster)
		result, err := validateCAConfig(context.Background(), secConfig, cluster)
		require.NoError(t, err, valid.description)

		// ensure that the cluster was not mutated
		require.Equal(t, valid.rootCA, cluster.RootCA)

		// Because join tokens are random, we can't predict exactly what it is, so this needs to be manually checked
		if valid.expectJoinTokenChange {
			require.NotEmpty(t, result.JoinTokens, valid.rootCA.JoinTokens, valid.description)
		} else {
			require.Equal(t, result.JoinTokens, valid.rootCA.JoinTokens, valid.description)
		}
		result.JoinTokens = valid.expectRootCA.JoinTokens

		// If a cross-signed certificates is generated, we cant know what it is ahead of time.  All we can do is check that it's
		// correctly generated.
		if valid.expectGeneratedCross || valid.expectGeneratedRootRotation { // both generate cross signed certs
			require.NotNil(t, result.RootRotation, valid.description)
			require.NotEmpty(t, result.RootRotation.CrossSignedCACert, valid.description)

			// make sure the cross-signed cert is signed by the current root CA (and not an intermediate, if a root rotation is in progress)
			parsedCross, err := helpers.ParseCertificatePEM(result.RootRotation.CrossSignedCACert) // there should just be one
			require.NoError(t, err)
			_, err = parsedCross.Verify(x509.VerifyOptions{Roots: localRootCA.Pool})
			require.NoError(t, err, valid.description)

			// if we are expecting generated certs or root rotation, we can expect the expected root CA has a root rotation
			result.RootRotation.CrossSignedCACert = valid.expectRootCA.RootRotation.CrossSignedCACert
		}

		// If a root rotation cert is generated, we can't assert what the cert and key are.  So if we expect it to be generated,
		// just assert that the value has changed.
		if valid.expectGeneratedRootRotation {
			require.NotNil(t, result.RootRotation, valid.description)
			require.NotEqual(t, valid.rootCA.RootRotation, result.RootRotation, valid.description)
			result.RootRotation = valid.expectRootCA.RootRotation
		}

		require.Equal(t, result, &valid.expectRootCA, valid.description)
	}
}

func TestValidateCAConfigValidValues(t *testing.T) {
	t.Parallel()
	localRootCA, err := ca.NewRootCA(testutils.ECDSA256SHA256Cert, testutils.ECDSA256SHA256Cert, testutils.ECDSA256Key,
		ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	parsedCert, err := helpers.ParseCertificatePEM(testutils.ECDSA256SHA256Cert)
	require.NoError(t, err)
	parsedKey, err := helpers.ParsePrivateKeyPEM(testutils.ECDSA256Key)
	require.NoError(t, err)

	initialExternalRootCA := initialLocalRootCA
	initialExternalRootCA.CAKey = nil

	// set up 2 external CAs that can be contacted for signing
	tempdir, err := ioutil.TempDir("", "test-validate-CA")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	initExtServer, err := testutils.NewExternalSigningServer(localRootCA, tempdir)
	require.NoError(t, err)
	defer initExtServer.Stop()
	require.NoError(t, initExtServer.EnableCASigning())

	// we need to accept client certs from the original cert
	rotationRootCA, err := ca.NewRootCA(append(initialLocalRootCA.CACert, rotationCert...), rotationCert, rotationKey,
		ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	rotateExtServer, err := testutils.NewExternalSigningServer(rotationRootCA, tempdir)
	require.NoError(t, err)
	defer rotateExtServer.Stop()
	require.NoError(t, rotateExtServer.EnableCASigning())

	getExpectedRootCA := func(hasKey bool) api.RootCA {
		result := initialLocalRootCA
		result.LastForcedRotation = 5
		result.JoinTokens = api.JoinTokens{}
		if !hasKey {
			result.CAKey = nil
		}
		return result
	}
	getRootCAWithRotation := func(base api.RootCA, cert, key, cross []byte) api.RootCA {
		init := base
		init.RootRotation = &api.RootRotation{
			CACert:            cert,
			CAKey:             key,
			CrossSignedCACert: cross,
		}
		return init
	}

	// These require no rotation, because the cert is exactly the same.
	testcases := []*rootCARotationTestCase{
		{
			description: "same desired cert and key as current Root CA results in no root rotation",
			rootCA:      initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: uglifyOnePEM(initialLocalRootCA.CACert),
				SigningCAKey:  initialLocalRootCA.CAKey,
				ForceRotate:   5,
			},
			expectRootCA: getExpectedRootCA(true),
		},
		{
			description: "same desired cert as current Root CA but external->internal results in no root rotation and no key -> key",
			rootCA:      initialExternalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: uglifyOnePEM(initialLocalRootCA.CACert),
				SigningCAKey:  initialLocalRootCA.CAKey,
				ForceRotate:   5,
				ExternalCAs: []*api.ExternalCA{
					{
						URL: initExtServer.URL,
					},
				},
			},
			expectRootCA: getExpectedRootCA(true),
		},
		{
			description: "same desired cert as current Root CA but internal->external results in no root rotation and key -> no key",
			rootCA:      initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: initialLocalRootCA.CACert,
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    initExtServer.URL,
						CACert: uglifyOnePEM(initialLocalRootCA.CACert),
					},
				},
				ForceRotate: 5,
			},
			expectRootCA: getExpectedRootCA(false),
		},
	}
	runValidTestCases(t, testcases, &localRootCA)

	// These will abort root rotation because the desired cert is the same as the current RootCA cert
	crossSigned, err := localRootCA.CrossSignCACertificate(rotationCert)
	require.NoError(t, err)
	for _, testcase := range testcases {
		testcase.rootCA = getRootCAWithRotation(testcase.rootCA, rotationCert, rotationKey, crossSigned)
	}
	testcases[0].description = "same desired cert and key as current RootCA results in aborting root rotation"
	testcases[1].description = "same desired cert, even if external->internal, as current RootCA results in aborting root rotation and no key -> key"
	testcases[2].description = "same desired cert, even if internal->external, as current RootCA results in aborting root rotation and key -> no key"
	runValidTestCases(t, testcases, &localRootCA)

	// These will not change the root rotation because the desired cert is the same as the current to-be-rotated-to cert
	expectedBaseRootCA := getExpectedRootCA(true) // the main root CA expected will always have a signing key
	testcases = []*rootCARotationTestCase{
		{
			description: "same desired cert and key as current root rotation results in no change in root rotation",
			rootCA:      getRootCAWithRotation(initialLocalRootCA, rotationCert, rotationKey, crossSigned),
			caConfig: api.CAConfig{
				SigningCACert: testutils.ECDSACertChain[2],
				SigningCAKey:  testutils.ECDSACertChainKeys[2],
				ForceRotate:   5,
			},
			expectRootCA: getRootCAWithRotation(expectedBaseRootCA, rotationCert, rotationKey, crossSigned),
		},
		{
			description: "same desired cert as current root rotation but external->internal results minor change in root rotation (no key -> key)",
			rootCA:      getRootCAWithRotation(initialLocalRootCA, rotationCert, nil, crossSigned),
			caConfig: api.CAConfig{
				SigningCACert: testutils.ECDSACertChain[2],
				SigningCAKey:  testutils.ECDSACertChainKeys[2],
				ForceRotate:   5,
			},
			expectRootCA: getRootCAWithRotation(expectedBaseRootCA, rotationCert, rotationKey, crossSigned),
		},
		{
			description: "same desired cert as current root rotation but internal->external results minor change in root rotation (key -> no key)",
			rootCA:      getRootCAWithRotation(initialLocalRootCA, rotationCert, rotationKey, crossSigned),
			caConfig: api.CAConfig{
				SigningCACert: testutils.ECDSACertChain[2],
				ForceRotate:   5,
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    rotateExtServer.URL,
						CACert: append(testutils.ECDSACertChain[2], ' '),
					},
				},
			},
			expectRootCA: getRootCAWithRotation(expectedBaseRootCA, rotationCert, nil, crossSigned),
		},
	}
	runValidTestCases(t, testcases, &localRootCA)

	// These all require a new root rotation because the desired cert is different, even if it has the same key and/or subject as the current
	// cert or the current-to-be-rotated cert.
	renewedInitialCert, err := initca.RenewFromSigner(parsedCert, parsedKey)
	require.NoError(t, err)
	parsedRotationCert, err := helpers.ParseCertificatePEM(rotationCert)
	require.NoError(t, err)
	parsedRotationKey, err := helpers.ParsePrivateKeyPEM(rotationKey)
	require.NoError(t, err)
	renewedRotationCert, err := initca.RenewFromSigner(parsedRotationCert, parsedRotationKey)
	require.NoError(t, err)
	differentInitialCert, err := testutils.CreateCertFromSigner("otherRootCN", parsedKey)
	require.NoError(t, err)
	differentRootCA, err := ca.NewRootCA(append(initialLocalRootCA.CACert, differentInitialCert...), differentInitialCert,
		initialLocalRootCA.CAKey, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	differentExtServer, err := testutils.NewExternalSigningServer(differentRootCA, tempdir)
	require.NoError(t, err)
	defer differentExtServer.Stop()
	require.NoError(t, differentExtServer.EnableCASigning())
	testcases = []*rootCARotationTestCase{
		{
			description: "desired cert being a renewed current cert and key results in a root rotation because the cert has changed",
			rootCA:      initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: uglifyOnePEM(renewedInitialCert),
				SigningCAKey:  initialLocalRootCA.CAKey,
				ForceRotate:   5,
			},
			expectRootCA:         getRootCAWithRotation(expectedBaseRootCA, renewedInitialCert, initialLocalRootCA.CAKey, nil),
			expectGeneratedCross: true,
		},
		{
			description: "desired cert being a renewed current cert, external->internal results in a root rotation because the cert has changed",
			rootCA:      initialExternalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: uglifyOnePEM(renewedInitialCert),
				SigningCAKey:  initialLocalRootCA.CAKey,
				ForceRotate:   5,
				ExternalCAs: []*api.ExternalCA{
					{
						URL: initExtServer.URL,
					},
				},
			},
			expectRootCA:         getRootCAWithRotation(getExpectedRootCA(false), renewedInitialCert, initialLocalRootCA.CAKey, nil),
			expectGeneratedCross: true,
		},
		{
			description: "desired cert being a renewed current cert, internal->external results in a root rotation because the cert has changed",
			rootCA:      initialLocalRootCA,
			caConfig: api.CAConfig{
				SigningCACert: append([]byte("\n\n"), renewedInitialCert...),
				ForceRotate:   5,
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    initExtServer.URL,
						CACert: uglifyOnePEM(renewedInitialCert),
					},
				},
			},
			expectRootCA:         getRootCAWithRotation(expectedBaseRootCA, renewedInitialCert, nil, nil),
			expectGeneratedCross: true,
		},
		{
			description: "desired cert being a renewed rotation RootCA cert + rotation key results in replaced root rotation because the cert has changed",
			rootCA:      getRootCAWithRotation(initialLocalRootCA, rotationCert, rotationKey, crossSigned),
			caConfig: api.CAConfig{
				SigningCACert: uglifyOnePEM(renewedRotationCert),
				SigningCAKey:  rotationKey,
				ForceRotate:   5,
			},
			expectRootCA:         getRootCAWithRotation(expectedBaseRootCA, renewedRotationCert, rotationKey, nil),
			expectGeneratedCross: true,
		},
		{
			description: "desired cert being a different rotation rootCA cert results in replaced root rotation (only new external CA required, not old rotation external CA)",
			rootCA:      getRootCAWithRotation(initialLocalRootCA, rotationCert, nil, crossSigned),
			caConfig: api.CAConfig{
				SigningCACert: uglifyOnePEM(differentInitialCert),
				ForceRotate:   5,
				ExternalCAs: []*api.ExternalCA{
					{
						// we need a different external server, because otherwise the external server's cert will fail to validate
						// (not signed by the right cert - note that there's a bug in go 1.7 where this is not needed, because the
						// subject names of cert names aren't checked, but go 1.8 fixes this.)
						URL:    differentExtServer.URL,
						CACert: append([]byte("\n\t"), differentInitialCert...),
					},
				},
			},
			expectRootCA:         getRootCAWithRotation(expectedBaseRootCA, differentInitialCert, nil, nil),
			expectGeneratedCross: true,
		},
	}
	runValidTestCases(t, testcases, &localRootCA)

	// These require rotation because the cert and key are generated and hence completely different.
	testcases = []*rootCARotationTestCase{
		{
			description:                 "generating cert and key results in root rotation",
			rootCA:                      initialLocalRootCA,
			caConfig:                    api.CAConfig{ForceRotate: 5},
			expectRootCA:                getRootCAWithRotation(getExpectedRootCA(true), nil, nil, nil),
			expectGeneratedRootRotation: true,
		},
		{
			description: "generating cert for external->internal results in root rotation",
			rootCA:      initialExternalRootCA,
			caConfig: api.CAConfig{
				ForceRotate: 5,
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    initExtServer.URL,
						CACert: uglifyOnePEM(initialExternalRootCA.CACert),
					},
				},
			},
			expectRootCA:                getRootCAWithRotation(getExpectedRootCA(false), nil, nil, nil),
			expectGeneratedRootRotation: true,
		},
		{
			description:                 "generating cert and key results in replacing root rotation",
			rootCA:                      getRootCAWithRotation(initialLocalRootCA, rotationCert, rotationKey, crossSigned),
			caConfig:                    api.CAConfig{ForceRotate: 5},
			expectRootCA:                getRootCAWithRotation(getExpectedRootCA(true), nil, nil, nil),
			expectGeneratedRootRotation: true,
		},
		{
			description:                 "generating cert and key results in replacing root rotation; external CAs required by old root rotation are no longer necessary",
			rootCA:                      getRootCAWithRotation(initialLocalRootCA, rotationCert, nil, crossSigned),
			caConfig:                    api.CAConfig{ForceRotate: 5},
			expectRootCA:                getRootCAWithRotation(getExpectedRootCA(true), nil, nil, nil),
			expectGeneratedRootRotation: true,
		},
	}
	runValidTestCases(t, testcases, &localRootCA)

	// These require no change at all because the force rotate value hasn't changed, and there is no desired cert specified
	testcases = []*rootCARotationTestCase{
		{
			description:  "no desired certificate specified, no force rotation: no change to internal signer root (which has no outstanding rotation)",
			rootCA:       initialLocalRootCA,
			expectRootCA: initialLocalRootCA,
		},
		{
			description: "no desired certificate specified, no force rotation: no change to external CA root (which has no outstanding rotation)",
			rootCA:      initialExternalRootCA,
			caConfig: api.CAConfig{
				ExternalCAs: []*api.ExternalCA{
					{
						URL:    initExtServer.URL,
						CACert: uglifyOnePEM(initialExternalRootCA.CACert),
					},
				},
			},
			expectRootCA: initialExternalRootCA,
		},
	}
	runValidTestCases(t, testcases, &localRootCA)

	for _, testcase := range testcases {
		testcase.rootCA = getRootCAWithRotation(testcase.rootCA, rotationCert, rotationKey, crossSigned)
		testcase.expectRootCA = testcase.rootCA
	}
	testcases[0].description = "no desired certificate specified, no force rotation: no change to internal signer root or to outstanding rotation"
	testcases[1].description = "no desired certificate specified, no force rotation: no change to external CA root or to outstanding rotation"
	runValidTestCases(t, testcases, &localRootCA)
}
