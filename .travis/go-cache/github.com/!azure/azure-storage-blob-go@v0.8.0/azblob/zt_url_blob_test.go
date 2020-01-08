package azblob_test

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io/ioutil"

	"github.com/Azure/azure-storage-blob-go/azblob"

	"bytes"

	"errors"
	"os"
	"strings"
	"time"

	chk "gopkg.in/check.v1" // go get gopkg.in/check.v1
)

// Copied from policy_unique_request_id.go
type uuid [16]byte

// The UUID reserved variants.
const (
	reservedNCS       byte = 0x80
	reservedRFC4122   byte = 0x40
	reservedMicrosoft byte = 0x20
	reservedFuture    byte = 0x00
)

func newUUID() (u uuid) {
	u = uuid{}
	// Set all bits to randomly (or pseudo-randomly) chosen values.
	rand.Read(u[:])
	u[8] = (u[8] | reservedRFC4122) & 0x7F // u.setVariant(ReservedRFC4122)

	var version byte = 4
	u[6] = (u[6] & 0xF) | (version << 4) // u.setVersion(4)
	return
}

func (u uuid) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func (s *aztestsSuite) TestCreateBlobURL(c *chk.C) {
	bsu := getBSU()
	containerURL, containerName := getContainerURL(c, bsu)
	testURL, testName := getBlockBlobURL(c, containerURL)

	parts := azblob.NewBlobURLParts(testURL.URL())
	c.Assert(parts.BlobName, chk.Equals, testName)
	c.Assert(parts.ContainerName, chk.Equals, containerName)

	correctURL := "https://" + os.Getenv("ACCOUNT_NAME") + ".blob.core.windows.net/" + containerName + "/" + testName
	temp := testURL.URL()
	c.Assert(temp.String(), chk.Equals, correctURL)
}

func (s *aztestsSuite) TestCreateBlobURLWithSnapshotAndSAS(c *chk.C) {
	bsu := getBSU()
	containerURL, containerName := getContainerURL(c, bsu)
	blobURL, blobName := getBlockBlobURL(c, containerURL)

	currentTime := time.Now().UTC()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	sasQueryParams, err := azblob.AccountSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    currentTime.Add(48 * time.Hour),
		Permissions:   azblob.AccountSASPermissions{Read: true, List: true}.String(),
		Services:      azblob.AccountSASServices{Blob: true}.String(),
		ResourceTypes: azblob.AccountSASResourceTypes{Container: true, Object: true}.String(),
	}.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	parts := azblob.NewBlobURLParts(blobURL.URL())
	parts.SAS = sasQueryParams
	parts.Snapshot = currentTime.Format(azblob.SnapshotTimeFormat)
	testURL := parts.URL()

	// The snapshot format string is taken from the snapshotTimeFormat value in parsing_urls.go. The field is not public, so
	// it is copied here
	correctURL := "https://" + os.Getenv("ACCOUNT_NAME") + ".blob.core.windows.net/" + containerName + "/" + blobName +
		"?" + "snapshot=" + currentTime.Format("2006-01-02T15:04:05.0000000Z07:00") + "&" + sasQueryParams.Encode()
	c.Assert(testURL.String(), chk.Equals, correctURL)
}

func (s *aztestsSuite) TestBlobWithNewPipeline(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)
	blobURL := containerURL.NewBlockBlobURL(blobPrefix)

	newBlobURL := blobURL.WithPipeline(testPipeline{})
	_, err := newBlobURL.GetBlockList(ctx, azblob.BlockListAll, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.NotNil)
	c.Assert(err.Error(), chk.Equals, testPipelineMessage)
}

func waitForCopy(c *chk.C, copyBlobURL azblob.BlockBlobURL, blobCopyResponse *azblob.BlobStartCopyFromURLResponse) {
	status := blobCopyResponse.CopyStatus()
	// Wait for the copy to finish. If the copy takes longer than a minute, we will fail
	start := time.Now()
	for status != azblob.CopyStatusSuccess {
		props, _ := copyBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
		status = props.CopyStatus()
		currentTime := time.Now()
		if currentTime.Sub(start) >= time.Minute {
			c.Fail()
		}
	}
}

func (s *aztestsSuite) TestBlobStartCopyDestEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)
	copyBlobURL, _ := getBlockBlobURL(c, containerURL)

	blobCopyResponse, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	waitForCopy(c, copyBlobURL, blobCopyResponse)

	resp, err := copyBlobURL.Download(ctx, 0, 20, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	// Read the blob data to verify the copy
	data, err := ioutil.ReadAll(resp.Response().Body)
	c.Assert(resp.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
	resp.Body(azblob.RetryReaderOptions{}).Close()
}

func (s *aztestsSuite) TestBlobStartCopyMetadata(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)
	copyBlobURL, _ := getBlockBlobURL(c, containerURL)

	resp, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	waitForCopy(c, copyBlobURL, resp)

	resp2, err := copyBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopyMetadataNil(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)
	copyBlobURL, _ := getBlockBlobURL(c, containerURL)

	// Have the destination start with metadata so we ensure the nil metadata passed later takes effect
	_, err := copyBlobURL.Upload(ctx, bytes.NewReader([]byte("data")), azblob.BlobHTTPHeaders{},
		basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	waitForCopy(c, copyBlobURL, resp)

	resp2, err := copyBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobStartCopyMetadataEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)
	copyBlobURL, _ := getBlockBlobURL(c, containerURL)

	// Have the destination start with metadata so we ensure the empty metadata passed later takes effect
	_, err := copyBlobURL.Upload(ctx, bytes.NewReader([]byte("data")), azblob.BlobHTTPHeaders{},
		basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), azblob.Metadata{}, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	waitForCopy(c, copyBlobURL, resp)

	resp2, err := copyBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobStartCopyMetadataInvalidField(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)
	copyBlobURL, _ := getBlockBlobURL(c, containerURL)

	_, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), azblob.Metadata{"I nvalid.": "bar"}, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.NotNil)
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)
}

func (s *aztestsSuite) TestBlobStartCopySourceNonExistant(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getBlockBlobURL(c, containerURL)
	copyBlobURL, _ := getBlockBlobURL(c, containerURL)

	_, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeBlobNotFound)
}

func (s *aztestsSuite) TestBlobStartCopySourcePrivate(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	bsu2, err := getAlternateBSU()
	if err != nil {
		c.Skip(err.Error())
		return
	}
	copyContainerURL, _ := createNewContainer(c, bsu2)
	defer deleteContainer(c, copyContainerURL)
	copyBlobURL, _ := getBlockBlobURL(c, copyContainerURL)

	if bsu.String() == bsu2.String() {
		c.Skip("Test not valid because primary and secondary accounts are the same")
	}
	_, err = copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeCannotVerifyCopySource)
}

func (s *aztestsSuite) TestBlobStartCopyUsingSASSrc(c *chk.C) {
	bsu := getBSU()
	containerURL, containerName := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)
	blobURL, blobName := createNewBlockBlob(c, containerURL)

	// Create sas values for the source blob
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	serviceSASValues := azblob.BlobSASSignatureValues{StartTime: time.Now().Add(-1 * time.Hour).UTC(),
		ExpiryTime: time.Now().Add(time.Hour).UTC(), Permissions: azblob.BlobSASPermissions{Read: true, Write: true}.String(),
		ContainerName: containerName, BlobName: blobName}
	queryParams, err := serviceSASValues.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	// Create URLs to the destination blob with sas parameters
	sasURL := blobURL.URL()
	sasURL.RawQuery = queryParams.Encode()

	// Create a new container for the destination
	bsu2, err := getAlternateBSU()
	if err != nil {
		c.Skip(err.Error())
		return
	}
	copyContainerURL, _ := createNewContainer(c, bsu2)
	defer deleteContainer(c, copyContainerURL)
	copyBlobURL, _ := getBlockBlobURL(c, copyContainerURL)

	resp, err := copyBlobURL.StartCopyFromURL(ctx, sasURL, nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	waitForCopy(c, copyBlobURL, resp)

	resp2, err := copyBlobURL.Download(ctx, 0, int64(len(blockBlobDefaultData)), azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	data, err := ioutil.ReadAll(resp2.Response().Body)
	c.Assert(resp2.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
	resp2.Body(azblob.RetryReaderOptions{}).Close()
}

func (s *aztestsSuite) TestBlobStartCopyUsingSASDest(c *chk.C) {
	bsu := getBSU()
	containerURL, containerName := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)
	blobURL, blobName := createNewBlockBlob(c, containerURL)
	_ = blobURL

	// Generate SAS on the source
	serviceSASValues := azblob.BlobSASSignatureValues{ExpiryTime: time.Now().Add(time.Hour).UTC(),
		Permissions: azblob.BlobSASPermissions{Read: true, Write: true, Create: true}.String(), ContainerName: containerName, BlobName: blobName}
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	queryParams, err := serviceSASValues.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	// Create destination container
	bsu2, err := getAlternateBSU()
	if err != nil {
		c.Skip(err.Error())
		return
	}

	copyContainerURL, copyContainerName := createNewContainer(c, bsu2)
	defer deleteContainer(c, copyContainerURL)
	copyBlobURL, copyBlobName := getBlockBlobURL(c, copyContainerURL)

	// Generate Sas for the destination
	credential, err = getGenericCredential("SECONDARY_")
	if err != nil {
		c.Fatal("Invalid secondary credential")
	}
	copyServiceSASvalues := azblob.BlobSASSignatureValues{StartTime: time.Now().Add(-1 * time.Hour).UTC(),
		ExpiryTime: time.Now().Add(time.Hour).UTC(), Permissions: azblob.BlobSASPermissions{Read: true, Write: true}.String(),
		ContainerName: copyContainerName, BlobName: copyBlobName}
	copyQueryParams, err := copyServiceSASvalues.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	// Generate anonymous URL to destination with SAS
	anonURL := bsu2.URL()
	anonURL.RawQuery = copyQueryParams.Encode()
	anonPipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	anonBSU := azblob.NewServiceURL(anonURL, anonPipeline)
	anonContainerURL := anonBSU.NewContainerURL(copyContainerName)
	anonBlobURL := anonContainerURL.NewBlockBlobURL(copyBlobName)

	// Apply sas to source
	srcBlobWithSasURL := blobURL.URL()
	srcBlobWithSasURL.RawQuery = queryParams.Encode()

	resp, err := anonBlobURL.StartCopyFromURL(ctx, srcBlobWithSasURL, nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	// Allow copy to happen
	waitForCopy(c, anonBlobURL, resp)

	resp2, err := copyBlobURL.Download(ctx, 0, int64(len(blockBlobDefaultData)), azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	data, err := ioutil.ReadAll(resp2.Response().Body)
	_, err = resp2.Body(azblob.RetryReaderOptions{}).Read(data)
	c.Assert(resp2.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
	resp2.Body(azblob.RetryReaderOptions{}).Close()
}

func (s *aztestsSuite) TestBlobStartCopySourceIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{IfModifiedSince: currentTime},
		azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil,
		azblob.ModifiedAccessConditions{IfModifiedSince: currentTime},
		azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeSourceConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime},
		azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil,
		azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime},
		azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeSourceConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	etag := resp.ETag()

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err = destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{IfMatch: etag},
		azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp2, err := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{IfMatch: "a"},
		azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeSourceConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{IfNoneMatch: "a"},
		azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp2, err := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopySourceIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	etag := resp.ETag()

	destBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err = destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil,
		azblob.ModifiedAccessConditions{IfNoneMatch: etag},
		azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeSourceConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL) // The blob must exist to have a last-modified time
	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	resp, err := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL)
	currentTime := getRelativeTimeGMT(10)

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil,
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeTargetConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL)
	currentTime := getRelativeTimeGMT(10)

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	resp, err := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)
	destBlobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil,
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeTargetConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL)
	resp, _ := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata,
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: etag}})
	c.Assert(err, chk.IsNil)

	resp, err = destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL)
	resp, _ := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()

	destBlobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{}) // SetMetadata chances the blob's etag

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: etag}})
	validateStorageError(c, err, azblob.ServiceCodeTargetConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL)
	resp, _ := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()

	destBlobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{}) // SetMetadata chances the blob's etag

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), basicMetadata, azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: etag}})
	c.Assert(err, chk.IsNil)

	resp, err = destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobStartCopyDestIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	destBlobURL, _ := createNewBlockBlob(c, containerURL)
	resp, _ := destBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()

	_, err := destBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: etag}})
	validateStorageError(c, err, azblob.ServiceCodeTargetConditionNotMet)
}

func (s *aztestsSuite) TestBlobAbortCopyInProgress(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getBlockBlobURL(c, containerURL)

	// Create a large blob that takes time to copy
	blobSize := 8 * 1024 * 1024
	blobData := make([]byte, blobSize, blobSize)
	for i := range blobData {
		blobData[i] = byte('a' + i%26)
	}
	_, err := blobURL.Upload(ctx, bytes.NewReader(blobData), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, nil, azblob.ContainerAccessConditions{}) // So that we don't have to create a SAS

	// Must copy across accounts so it takes time to copy
	bsu2, err := getAlternateBSU()
	if err != nil {
		c.Skip(err.Error())
		return
	}

	copyContainerURL, _ := createNewContainer(c, bsu2)
	copyBlobURL, _ := getBlockBlobURL(c, copyContainerURL)

	defer deleteContainer(c, copyContainerURL)

	resp, err := copyBlobURL.StartCopyFromURL(ctx, blobURL.URL(), nil, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.CopyStatus(), chk.Equals, azblob.CopyStatusPending)

	_, err = copyBlobURL.AbortCopyFromURL(ctx, resp.CopyID(), azblob.LeaseAccessConditions{})
	if err != nil {
		// If the error is nil, the test continues as normal.
		// If the error is not nil, we want to check if it's because the copy is finished and send a message indicating this.
		validateStorageError(c, err, azblob.ServiceCodeNoPendingCopyOperation)
		c.Error("The test failed because the copy completed because it was aborted")
	}

	resp2, _ := copyBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(resp2.CopyStatus(), chk.Equals, azblob.CopyStatusAborted)
}

func (s *aztestsSuite) TestBlobAbortCopyNoCopyStarted(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	copyBlobURL, _ := getBlockBlobURL(c, containerURL)
	_, err := copyBlobURL.AbortCopyFromURL(ctx, "copynotstarted", azblob.LeaseAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeInvalidQueryParameterValue)
}

func (s *aztestsSuite) TestBlobSnapshotMetadata(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.CreateSnapshot(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	// Since metadata is specified on the snapshot, the snapshot should have its own metadata different from the (empty) metadata on the source
	snapshotURL := blobURL.WithSnapshot(resp.Snapshot())
	resp2, err := snapshotURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobSnapshotMetadataEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.CreateSnapshot(ctx, azblob.Metadata{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	// In this case, because no metadata was specified, it should copy the basicMetadata from the source
	snapshotURL := blobURL.WithSnapshot(resp.Snapshot())
	resp2, err := snapshotURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobSnapshotMetadataNil(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	snapshotURL := blobURL.WithSnapshot(resp.Snapshot())
	resp2, err := snapshotURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobSnapshotMetadataInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, azblob.Metadata{"Invalid Field!": "value"}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.NotNil)
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)
}

func (s *aztestsSuite) TestBlobSnapshotBlobNotExist(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getBlockBlobURL(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeBlobNotFound)
}

func (s *aztestsSuite) TestBlobSnapshotOfSnapshot(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	snapshotURL := blobURL.WithSnapshot(time.Now().UTC().Format(azblob.SnapshotTimeFormat))
	// The library allows the server to handle the snapshot of snapshot error
	_, err := snapshotURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeInvalidQueryParameterValue)
}

func (s *aztestsSuite) TestBlobSnapshotIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Snapshot() != "", chk.Equals, true) // i.e. The snapshot time is not zero. If the service gives us back a snapshot time, it successfully created a snapshot
}

func (s *aztestsSuite) TestBlobSnapshotIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSnapshotIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	resp, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Snapshot() == "", chk.Equals, false)
}

func (s *aztestsSuite) TestBlobSnapshotIfUnmodifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSnapshotIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	resp2, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.Snapshot() == "", chk.Equals, false)
}

func (s *aztestsSuite) TestBlobSnapshotIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: "garbage"}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSnapshotIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: "garbage"}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Snapshot() == "", chk.Equals, false)
}

func (s *aztestsSuite) TestBlobSnapshotIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	_, err = blobURL.CreateSnapshot(ctx, nil,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDownloadDataNonExistantBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getBlockBlobURL(c, containerURL)

	_, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	validateStorageError(c, err, azblob.ServiceCodeBlobNotFound)
}

func (s *aztestsSuite) TestBlobDownloadDataNegativeOffset(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Download(ctx, -1, 0, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestBlobDownloadDataOffsetOutOfRange(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Download(ctx, int64(len(blockBlobDefaultData)), azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	validateStorageError(c, err, azblob.ServiceCodeInvalidRange)
}

func (s *aztestsSuite) TestBlobDownloadDataCountNegative(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Download(ctx, 0, -2, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestBlobDownloadDataCountZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	// Specifying a count of 0 results in the value being ignored
	data, err := ioutil.ReadAll(resp.Response().Body)
	c.Assert(err, chk.IsNil)
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
}

func (s *aztestsSuite) TestBlobDownloadDataCountExact(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.Download(ctx, 0, int64(len(blockBlobDefaultData)), azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	data, err := ioutil.ReadAll(resp.Response().Body)
	c.Assert(err, chk.IsNil)
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
}

func (s *aztestsSuite) TestBlobDownloadDataCountOutOfRange(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.Download(ctx, 0, int64(len(blockBlobDefaultData))*2, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	data, err := ioutil.ReadAll(resp.Response().Body)
	c.Assert(err, chk.IsNil)
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
}

func (s *aztestsSuite) TestBlobDownloadDataEmptyRangeStruct(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)

	data, err := ioutil.ReadAll(resp.Response().Body)
	c.Assert(err, chk.IsNil)
	c.Assert(string(data), chk.Equals, blockBlobDefaultData)
}

func (s *aztestsSuite) TestBlobDownloadDataContentMD5(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.Download(ctx, 10, 3, azblob.BlobAccessConditions{}, true)
	c.Assert(err, chk.IsNil)
	mdf := md5.Sum([]byte(blockBlobDefaultData)[10:13])
	c.Assert(resp.ContentMD5(), chk.DeepEquals, mdf[:])
}

func (s *aztestsSuite) TestBlobDownloadDataIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}}, false)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
}

func (s *aztestsSuite) TestBlobDownloadDataIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}}, false)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // The server does not return the error in the body even though it is a GET
}

func (s *aztestsSuite) TestBlobDownloadDataIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	resp, err := blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, false)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
}

func (s *aztestsSuite) TestBlobDownloadDataIfUnmodifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, false)
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDownloadDataIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	etag := resp.ETag()

	resp2, err := blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: etag}}, false)
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
}

func (s *aztestsSuite) TestBlobDownloadDataIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	etag := resp.ETag()

	blobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{})

	_, err = blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: etag}}, false)
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDownloadDataIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	etag := resp.ETag()

	blobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{})

	resp2, err := blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: etag}}, false)
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.ContentLength(), chk.Equals, int64(len(blockBlobDefaultData)))
}

func (s *aztestsSuite) TestBlobDownloadDataIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	etag := resp.ETag()

	_, err = blobURL.Download(ctx, 0, 0,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: etag}}, false)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // The server does not return the error in the body even though it is a GET
}

func (s *aztestsSuite) TestBlobDeleteNonExistant(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getBlockBlobURL(c, containerURL)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeBlobNotFound)
}

func (s *aztestsSuite) TestBlobDeleteSnapshot(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	snapshotURL := blobURL.WithSnapshot(resp.Snapshot())

	_, err = snapshotURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	validateBlobDeleted(c, snapshotURL)
}

func (s *aztestsSuite) TestBlobDeleteSnapshotsInclude(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, _ := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Snapshots: true}})
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobDeleteSnapshotsOnly(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionOnly, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, _ := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Snapshots: true}})
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 1)
	c.Assert(resp.Segment.BlobItems[0].Snapshot == "", chk.Equals, true)
}

func (s *aztestsSuite) TestBlobDeleteSnapshotsNoneWithSnapshots(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.CreateSnapshot(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeSnapshotsPresent)
}

func validateBlobDeleted(c *chk.C, blobURL azblob.BlockBlobURL) {
	_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError) // Delete blob is a HEAD request and does not return a ServiceCode in the body
	c.Assert(serr.Response().StatusCode, chk.Equals, 404)
}

func (s *aztestsSuite) TestBlobDeleteIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateBlobDeleted(c, blobURL)
}

func (s *aztestsSuite) TestBlobDeleteIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDeleteIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateBlobDeleted(c, blobURL)
}

func (s *aztestsSuite) TestBlobDeleteIfUnmodifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDeleteIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: etag}})
	c.Assert(err, chk.IsNil)

	validateBlobDeleted(c, blobURL)
}

func (s *aztestsSuite) TestBlobDeleteIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()
	blobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{})

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: etag}})

	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDeleteIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()
	blobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{})

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: etag}})
	c.Assert(err, chk.IsNil)

	validateBlobDeleted(c, blobURL)
}

func (s *aztestsSuite) TestBlobDeleteIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	etag := resp.ETag()

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: etag}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	currentTime := getRelativeTimeGMT(10)

	_, err = blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // No service code returned for a HEAD
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	currentTime := getRelativeTimeGMT(10)

	resp, err := blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 412)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp2, err := blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobGetPropsOnMissingBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL := containerURL.NewBlobURL("MISSING")

	_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 404)
	c.Assert(serr.ServiceCode(), chk.Equals, azblob.ServiceCodeBlobNotFound)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: azblob.ETag("garbage")}})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 412)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: azblob.ETag("garbage")}})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobGetPropsAndMetadataIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.GetProperties(ctx,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304)
}

func (s *aztestsSuite) TestBlobSetPropertiesBasic(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetHTTPHeaders(ctx, basicHeaders, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	h := resp.NewHTTPHeaders()
	c.Assert(h, chk.DeepEquals, basicHeaders)
}

func (s *aztestsSuite) TestBlobSetPropertiesEmptyValue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentType: "my_type"}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContentType(), chk.Equals, "")
}

func validatePropertiesSet(c *chk.C, blobURL azblob.BlockBlobURL, disposition string) {
	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContentDisposition(), chk.Equals, disposition)
}

func (s *aztestsSuite) TestBlobSetPropertiesIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validatePropertiesSet(c, blobURL, "my_disposition")
}

func (s *aztestsSuite) TestBlobSetPropertiesIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetPropertiesIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validatePropertiesSet(c, blobURL, "my_disposition")
}

func (s *aztestsSuite) TestBlobSetPropertiesIfUnmodifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetPropertiesIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)

	validatePropertiesSet(c, blobURL, "my_disposition")
}

func (s *aztestsSuite) TestBlobSetPropertiesIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: azblob.ETag("garbage")}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetPropertiesIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: azblob.ETag("garbage")}})
	c.Assert(err, chk.IsNil)

	validatePropertiesSet(c, blobURL, "my_disposition")
}

func (s *aztestsSuite) TestBlobSetPropertiesIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetHTTPHeaders(ctx, azblob.BlobHTTPHeaders{ContentDisposition: "my_disposition"},
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetMetadataNil(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, azblob.Metadata{"not": "nil"}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetMetadata(ctx, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobSetMetadataEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, azblob.Metadata{"not": "nil"}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetMetadata(ctx, azblob.Metadata{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobSetMetadataInvalidField(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, azblob.Metadata{"Invalid field!": "value"}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.NotNil)
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)
}

func validateMetadataSet(c *chk.C, blobURL azblob.BlockBlobURL) {
	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobSetMetadataIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateMetadataSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetMetadataIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetMetadataIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateMetadataSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetMetadataIfUnmodifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetMetadataIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)

	validateMetadataSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetMetadataIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: azblob.ETag("garbage")}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetMetadataIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: azblob.ETag("garbage")}})
	c.Assert(err, chk.IsNil)

	validateMetadataSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetMetadataIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = blobURL.SetMetadata(ctx, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func testBlobsUndeleteImpl(c *chk.C, bsu azblob.ServiceURL) error {
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil) // This call will not have errors related to slow update of service properties, so we assert.

	_, err = blobURL.Undelete(ctx)
	if err != nil { // We want to give the wrapper method a chance to check if it was an error related to the service properties update.
		return err
	}

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	if err != nil {
		return errors.New(string(err.(azblob.StorageError).ServiceCode()))
	}
	c.Assert(resp.BlobType(), chk.Equals, azblob.BlobBlockBlob) // We could check any property. This is just to double check it was undeleted.
	return nil
}

func (s *aztestsSuite) TestBlobsUndelete(c *chk.C) {
	bsu := getBSU()

	runTestRequiringServiceProperties(c, bsu, string(azblob.ServiceCodeBlobNotFound), enableSoftDelete, testBlobsUndeleteImpl, disableSoftDelete)
}

func setAndCheckBlobTier(c *chk.C, containerURL azblob.ContainerURL, blobURL azblob.BlobURL, tier azblob.AccessTierType) {
	_, err := blobURL.SetTier(ctx, tier, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.AccessTier(), chk.Equals, string(tier))

	resp2, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.Segment.BlobItems[0].Properties.AccessTier, chk.Equals, tier)
}

func (s *aztestsSuite) TestBlobSetTierAllTiers(c *chk.C) {
	bsu, err := getBlobStorageBSU()
	if err != nil {
		c.Skip(err.Error())
	}
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	setAndCheckBlobTier(c, containerURL, blobURL.BlobURL, azblob.AccessTierHot)
	setAndCheckBlobTier(c, containerURL, blobURL.BlobURL, azblob.AccessTierCool)
	setAndCheckBlobTier(c, containerURL, blobURL.BlobURL, azblob.AccessTierArchive)

	bsu, err = getPremiumBSU()
	if err != nil {
		c.Skip(err.Error())
	}

	containerURL, _ = createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	pageBlobURL, _ := createNewPageBlob(c, containerURL)

	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP4)
	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP6)
	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP10)
	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP20)
	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP30)
	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP40)
	setAndCheckBlobTier(c, containerURL, pageBlobURL.BlobURL, azblob.AccessTierP50)
}

func (s *aztestsSuite) TestBlobTierInferred(c *chk.C) {
	bsu, err := getPremiumBSU()
	if err != nil {
		c.Skip(err.Error())
	}

	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.AccessTierInferred(), chk.Equals, "true")

	resp2, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.Segment.BlobItems[0].Properties.AccessTierInferred, chk.NotNil)
	c.Assert(resp2.Segment.BlobItems[0].Properties.AccessTier, chk.Not(chk.Equals), "")

	_, err = blobURL.SetTier(ctx, azblob.AccessTierP4, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err = blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.AccessTierInferred(), chk.Equals, "")

	resp2, err = containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.Segment.BlobItems[0].Properties.AccessTierInferred, chk.IsNil) // AccessTierInferred never returned if false
}

func (s *aztestsSuite) TestBlobArchiveStatus(c *chk.C) {
	bsu, err := getBlobStorageBSU()
	if err != nil {
		c.Skip(err.Error())
	}

	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err = blobURL.SetTier(ctx, azblob.AccessTierArchive, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.SetTier(ctx, azblob.AccessTierCool, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ArchiveStatus(), chk.Equals, string(azblob.ArchiveStatusRehydratePendingToCool))

	resp2, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.Segment.BlobItems[0].Properties.ArchiveStatus, chk.Equals, azblob.ArchiveStatusRehydratePendingToCool)

	// delete first blob
	_, err = blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	blobURL, _ = createNewBlockBlob(c, containerURL)

	_, err = blobURL.SetTier(ctx, azblob.AccessTierArchive, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.SetTier(ctx, azblob.AccessTierHot, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err = blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ArchiveStatus(), chk.Equals, string(azblob.ArchiveStatusRehydratePendingToHot))

	resp2, err = containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.Segment.BlobItems[0].Properties.ArchiveStatus, chk.Equals, azblob.ArchiveStatusRehydratePendingToHot)
}

func (s *aztestsSuite) TestBlobTierInvalidValue(c *chk.C) {
	bsu, err := getBlobStorageBSU()
	if err != nil {
		c.Skip(err.Error())
	}

	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err = blobURL.SetTier(ctx, azblob.AccessTierType("garbage"), azblob.LeaseAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeInvalidHeaderValue)
}
