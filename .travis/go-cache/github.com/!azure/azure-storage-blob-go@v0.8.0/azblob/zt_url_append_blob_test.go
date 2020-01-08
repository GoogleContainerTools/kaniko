package azblob_test

import (
	"context"
	"io/ioutil"
	"time"

	"crypto/md5"

	"bytes"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	chk "gopkg.in/check.v1" // go get gopkg.in/check.v1
)

func (s *aztestsSuite) TestAppendBlock(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	blob := container.NewAppendBlobURL(generateBlobName())

	resp, err := blob.Create(context.Background(), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.StatusCode(), chk.Equals, 201)

	appendResp, err := blob.AppendBlock(context.Background(), getReaderToRandomBytes(1024), azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	c.Assert(appendResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(appendResp.BlobAppendOffset(), chk.Equals, "0")
	c.Assert(appendResp.BlobCommittedBlockCount(), chk.Equals, int32(1))
	c.Assert(appendResp.ETag(), chk.Not(chk.Equals), azblob.ETagNone)
	c.Assert(appendResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(appendResp.ContentMD5(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Date().IsZero(), chk.Equals, false)

	appendResp, err = blob.AppendBlock(context.Background(), getReaderToRandomBytes(1024), azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	c.Assert(appendResp.BlobAppendOffset(), chk.Equals, "1024")
	c.Assert(appendResp.BlobCommittedBlockCount(), chk.Equals, int32(2))
}

func (s *aztestsSuite) TestAppendBlockWithMD5(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	// set up blob to test
	blob := container.NewAppendBlobURL(generateBlobName())
	resp, err := blob.Create(context.Background(), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.StatusCode(), chk.Equals, 201)

	// test append block with valid MD5 value
	readerToBody, body := getRandomDataAndReader(1024)
	md5Value := md5.Sum(body)
	appendResp, err := blob.AppendBlock(context.Background(), readerToBody, azblob.AppendBlobAccessConditions{}, md5Value[:])
	c.Assert(err, chk.IsNil)
	c.Assert(appendResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(appendResp.BlobAppendOffset(), chk.Equals, "0")
	c.Assert(appendResp.BlobCommittedBlockCount(), chk.Equals, int32(1))
	c.Assert(appendResp.ETag(), chk.Not(chk.Equals), azblob.ETagNone)
	c.Assert(appendResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(appendResp.ContentMD5(), chk.DeepEquals, md5Value[:])
	c.Assert(appendResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Date().IsZero(), chk.Equals, false)

	// test append block with bad MD5 value
	readerToBody, body = getRandomDataAndReader(1024)
	_, badMD5 := getRandomDataAndReader(16)
	appendResp, err = blob.AppendBlock(context.Background(), readerToBody, azblob.AppendBlobAccessConditions{}, badMD5[:])
	validateStorageError(c, err, azblob.ServiceCodeMd5Mismatch)
}

func (s *aztestsSuite) TestAppendBlockFromURL(c *chk.C) {
	bsu := getBSU()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	testSize := 4 * 1024 * 1024 // 4MB
	r, sourceData := getRandomDataAndReader(testSize)
	ctx := context.Background() // Use default Background context
	srcBlob := container.NewAppendBlobURL(generateName("appendsrc"))
	destBlob := container.NewAppendBlobURL(generateName("appenddest"))

	// Prepare source blob for copy.
	cResp1, err := srcBlob.Create(context.Background(), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(cResp1.StatusCode(), chk.Equals, 201)
	appendResp, err := srcBlob.AppendBlock(context.Background(), r, azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	c.Assert(appendResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(appendResp.BlobAppendOffset(), chk.Equals, "0")
	c.Assert(appendResp.BlobCommittedBlockCount(), chk.Equals, int32(1))
	c.Assert(appendResp.ETag(), chk.Not(chk.Equals), azblob.ETagNone)
	c.Assert(appendResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(appendResp.ContentMD5(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Date().IsZero(), chk.Equals, false)

	// Get source blob URL with SAS for AppendBlockFromURL.
	srcBlobParts := azblob.NewBlobURLParts(srcBlob.URL())

	srcBlobParts.SAS, err = azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,              // Users MUST use HTTPS (not HTTP)
		ExpiryTime:    time.Now().UTC().Add(48 * time.Hour), // 48-hours before expiration
		ContainerName: srcBlobParts.ContainerName,
		BlobName:      srcBlobParts.BlobName,
		Permissions:   azblob.BlobSASPermissions{Read: true}.String(),
	}.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	srcBlobURLWithSAS := srcBlobParts.URL()

	// Append block from URL.
	cResp2, err := destBlob.Create(context.Background(), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(cResp2.StatusCode(), chk.Equals, 201)
	appendFromURLResp, err := destBlob.AppendBlockFromURL(ctx, srcBlobURLWithSAS, 0, int64(testSize), azblob.AppendBlobAccessConditions{}, azblob.ModifiedAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	c.Assert(appendFromURLResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(appendFromURLResp.BlobAppendOffset(), chk.Equals, "0")
	c.Assert(appendFromURLResp.BlobCommittedBlockCount(), chk.Equals, int32(1))
	c.Assert(appendFromURLResp.ETag(), chk.Not(chk.Equals), azblob.ETagNone)
	c.Assert(appendFromURLResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(appendFromURLResp.ContentMD5(), chk.Not(chk.Equals), "")
	c.Assert(appendFromURLResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(appendFromURLResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(appendFromURLResp.Date().IsZero(), chk.Equals, false)

	// Check data integrity through downloading.
	downloadResp, err := destBlob.BlobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)
	destData, err := ioutil.ReadAll(downloadResp.Body(azblob.RetryReaderOptions{}))
	c.Assert(err, chk.IsNil)
	c.Assert(destData, chk.DeepEquals, sourceData)
}

func (s *aztestsSuite) TestAppendBlockFromURLWithMD5(c *chk.C) {
	bsu := getBSU()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	testSize := 4 * 1024 * 1024 // 4MB
	r, sourceData := getRandomDataAndReader(testSize)
	md5Value := md5.Sum(sourceData)
	ctx := context.Background() // Use default Background context
	srcBlob := container.NewAppendBlobURL(generateName("appendsrc"))
	destBlob := container.NewAppendBlobURL(generateName("appenddest"))

	// Prepare source blob for copy.
	cResp1, err := srcBlob.Create(context.Background(), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(cResp1.StatusCode(), chk.Equals, 201)
	appendResp, err := srcBlob.AppendBlock(context.Background(), r, azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	c.Assert(appendResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(appendResp.BlobAppendOffset(), chk.Equals, "0")
	c.Assert(appendResp.BlobCommittedBlockCount(), chk.Equals, int32(1))
	c.Assert(appendResp.ETag(), chk.Not(chk.Equals), azblob.ETagNone)
	c.Assert(appendResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(appendResp.ContentMD5(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(appendResp.Date().IsZero(), chk.Equals, false)

	// Get source blob URL with SAS for AppendBlockFromURL.
	srcBlobParts := azblob.NewBlobURLParts(srcBlob.URL())

	srcBlobParts.SAS, err = azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,              // Users MUST use HTTPS (not HTTP)
		ExpiryTime:    time.Now().UTC().Add(48 * time.Hour), // 48-hours before expiration
		ContainerName: srcBlobParts.ContainerName,
		BlobName:      srcBlobParts.BlobName,
		Permissions:   azblob.BlobSASPermissions{Read: true}.String(),
	}.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	srcBlobURLWithSAS := srcBlobParts.URL()

	// Append block from URL.
	cResp2, err := destBlob.Create(context.Background(), azblob.BlobHTTPHeaders{}, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(cResp2.StatusCode(), chk.Equals, 201)
	appendFromURLResp, err := destBlob.AppendBlockFromURL(ctx, srcBlobURLWithSAS, 0, int64(testSize), azblob.AppendBlobAccessConditions{}, azblob.ModifiedAccessConditions{}, md5Value[:])
	c.Assert(err, chk.IsNil)
	c.Assert(appendFromURLResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(appendFromURLResp.BlobAppendOffset(), chk.Equals, "0")
	c.Assert(appendFromURLResp.BlobCommittedBlockCount(), chk.Equals, int32(1))
	c.Assert(appendFromURLResp.ETag(), chk.Not(chk.Equals), azblob.ETagNone)
	c.Assert(appendFromURLResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(appendFromURLResp.ContentMD5(), chk.DeepEquals, md5Value[:])
	c.Assert(appendFromURLResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(appendFromURLResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(appendFromURLResp.Date().IsZero(), chk.Equals, false)

	// Check data integrity through downloading.
	downloadResp, err := destBlob.BlobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	c.Assert(err, chk.IsNil)
	destData, err := ioutil.ReadAll(downloadResp.Body(azblob.RetryReaderOptions{}))
	c.Assert(err, chk.IsNil)
	c.Assert(destData, chk.DeepEquals, sourceData)

	// Test append block from URL with bad MD5 value
	_, badMD5 := getRandomDataAndReader(16)
	_, err = destBlob.AppendBlockFromURL(ctx, srcBlobURLWithSAS, 0, int64(testSize), azblob.AppendBlobAccessConditions{}, azblob.ModifiedAccessConditions{}, badMD5)
	validateStorageError(c, err, azblob.ServiceCodeMd5Mismatch)
}

func (s *aztestsSuite) TestBlobCreateAppendMetadataNonEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getAppendBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobCreateAppendMetadataEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getAppendBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobCreateAppendMetadataInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getAppendBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, azblob.Metadata{"In valid!": "bar"}, azblob.BlobAccessConditions{})
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)
}

func (s *aztestsSuite) TestBlobCreateAppendHTTPHeaders(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getAppendBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, basicHeaders, nil, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	h := resp.NewHTTPHeaders()
	c.Assert(h, chk.DeepEquals, basicHeaders)
}

func validateAppendBlobPut(c *chk.C, blobURL azblob.AppendBlobURL) {
	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobCreateAppendIfModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateAppendBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreateAppendIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobCreateAppendIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateAppendBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreateAppendIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobCreateAppendIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)

	validateAppendBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreateAppendIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: azblob.ETag("garbage")}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobCreateAppendIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: azblob.ETag("garbage")}})
	c.Assert(err, chk.IsNil)

	validateAppendBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreateAppendIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	_, err := blobURL.Create(ctx, azblob.BlobHTTPHeaders{}, basicMetadata,
		azblob.BlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockNilBody(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, bytes.NewReader(nil), azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.NotNil)
	validateStorageError(c, err, azblob.ServiceCodeInvalidHeaderValue)
}

func (s *aztestsSuite) TestBlobAppendBlockEmptyBody(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(""), azblob.AppendBlobAccessConditions{}, nil)
	validateStorageError(c, err, azblob.ServiceCodeInvalidHeaderValue)
}

func (s *aztestsSuite) TestBlobAppendBlockNonExistantBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getAppendBlobURL(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData), azblob.AppendBlobAccessConditions{}, nil)
	validateStorageError(c, err, azblob.ServiceCodeBlobNotFound)
}

func validateBlockAppended(c *chk.C, blobURL azblob.AppendBlobURL, expectedSize int) {
	resp, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContentLength(), chk.Equals, int64(expectedSize))
}

func (s *aztestsSuite) TestBlobAppendBlockIfModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)
	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}}, nil)
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)
	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)
	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, nil)
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: resp.ETag()}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: azblob.ETag("garbage")}}, nil)
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: azblob.ETag("garbage")}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfNoneMatch: resp.ETag()}}, nil)
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockIfAppendPositionMatchTrueNegOne(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfAppendPositionEqual: -1}}, nil) // This will cause the library to set the value of the header to 0
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfAppendPositionMatchZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData), azblob.AppendBlobAccessConditions{}, nil) // The position will not match, but the condition should be ignored
	c.Assert(err, chk.IsNil)
	_, err = blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfAppendPositionEqual: 0}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, 2*len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfAppendPositionMatchTrueNonZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData), azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	_, err = blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfAppendPositionEqual: int64(len(blockBlobDefaultData))}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData)*2)
}

func (s *aztestsSuite) TestBlobAppendBlockIfAppendPositionMatchFalseNegOne(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData), azblob.AppendBlobAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)
	_, err = blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfAppendPositionEqual: -1}}, nil) // This will cause the library to set the value of the header to 0
	validateStorageError(c, err, azblob.ServiceCodeAppendPositionConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockIfAppendPositionMatchFalseNonZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfAppendPositionEqual: 12}}, nil)
	validateStorageError(c, err, azblob.ServiceCodeAppendPositionConditionNotMet)
}

func (s *aztestsSuite) TestBlobAppendBlockIfMaxSizeTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfMaxSizeLessThanOrEqual: int64(len(blockBlobDefaultData) + 1)}}, nil)
	c.Assert(err, chk.IsNil)

	validateBlockAppended(c, blobURL, len(blockBlobDefaultData))
}

func (s *aztestsSuite) TestBlobAppendBlockIfMaxSizeFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewAppendBlob(c, containerURL)

	_, err := blobURL.AppendBlock(ctx, strings.NewReader(blockBlobDefaultData),
		azblob.AppendBlobAccessConditions{AppendPositionAccessConditions: azblob.AppendPositionAccessConditions{IfMaxSizeLessThanOrEqual: int64(len(blockBlobDefaultData) - 1)}}, nil)
	validateStorageError(c, err, azblob.ServiceCodeMaxBlobSizeConditionNotMet)
}
