package azblob_test

import (
	"context"
	"time"

	"bytes"

	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	chk "gopkg.in/check.v1" // go get gopkg.in/check.v1
)

func delContainer(c *chk.C, container azblob.ContainerURL) {
	resp, err := container.Delete(context.Background(), azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 202)
}

func (s *aztestsSuite) TestNewContainerURLValidName(c *chk.C) {
	bsu := getBSU()
	testURL := bsu.NewContainerURL(containerPrefix)

	correctURL := "https://" + os.Getenv("ACCOUNT_NAME") + ".blob.core.windows.net/" + containerPrefix
	temp := testURL.URL()
	c.Assert(temp.String(), chk.Equals, correctURL)
}

func (s *aztestsSuite) TestCreateRootContainerURL(c *chk.C) {
	bsu := getBSU()
	testURL := bsu.NewContainerURL(azblob.ContainerNameRoot)

	correctURL := "https://" + os.Getenv("ACCOUNT_NAME") + ".blob.core.windows.net/$root"
	temp := testURL.URL()
	c.Assert(temp.String(), chk.Equals, correctURL)
}

func (s *aztestsSuite) TestAccountWithPipeline(c *chk.C) {
	bsu := getBSU()
	bsu = bsu.WithPipeline(testPipeline{}) // testPipeline returns an identifying message as an error
	containerURL := bsu.NewContainerURL("name")

	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessBlob)

	c.Assert(err.Error(), chk.Equals, testPipelineMessage)
}

func (s *aztestsSuite) TestContainerCreateInvalidName(c *chk.C) {
	bsu := getBSU()
	containerURL := bsu.NewContainerURL("foo bar")

	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessBlob)

	validateStorageError(c, err, azblob.ServiceCodeInvalidResourceName)
}

func (s *aztestsSuite) TestContainerCreateEmptyName(c *chk.C) {
	bsu := getBSU()
	containerURL := bsu.NewContainerURL("")

	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessBlob)

	validateStorageError(c, err, azblob.ServiceCodeInvalidQueryParameterValue)
}

func (s *aztestsSuite) TestContainerCreateNameCollision(c *chk.C) {
	bsu := getBSU()
	containerURL, containerName := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	containerURL = bsu.NewContainerURL(containerName)
	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessBlob)

	validateStorageError(c, err, azblob.ServiceCodeContainerAlreadyExists)
}

func (s *aztestsSuite) TestContainerCreateInvalidMetadata(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Create(ctx, azblob.Metadata{"1 foo": "bar"}, azblob.PublicAccessBlob)

	c.Assert(err, chk.NotNil)
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)
}

func (s *aztestsSuite) TestContainerCreateNilMetadata(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Create(ctx, nil, azblob.PublicAccessBlob)
	defer deleteContainer(c, containerURL)
	c.Assert(err, chk.IsNil)

	response, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(response.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestContainerCreateEmptyMetadata(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessBlob)
	defer deleteContainer(c, containerURL)
	c.Assert(err, chk.IsNil)

	response, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(response.NewMetadata(), chk.HasLen, 0)
}

// Note that for all tests that create blobs, deleting the container also deletes any blobs within that container, thus we
// simply delete the whole container after the test

func (s *aztestsSuite) TestContainerCreateAccessContainer(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Create(ctx, nil, azblob.PublicAccessContainer)
	defer deleteContainer(c, containerURL)
	c.Assert(err, chk.IsNil)

	blobURL := containerURL.NewBlockBlobURL(blobPrefix)
	blobURL.Upload(ctx, bytes.NewReader([]byte("Content")), azblob.BlobHTTPHeaders{},
		basicMetadata, azblob.BlobAccessConditions{})

	// Anonymous enumeration should be valid with container access
	containerURL2 := azblob.NewContainerURL(containerURL.URL(), azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
	response, err := containerURL2.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(response.Segment.BlobItems[0].Name, chk.Equals, blobPrefix)

	// Getting blob data anonymously should still be valid with container access
	blobURL2 := containerURL2.NewBlockBlobURL(blobPrefix)
	resp, err := blobURL2.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestContainerCreateAccessBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Create(ctx, nil, azblob.PublicAccessBlob)
	defer deleteContainer(c, containerURL)
	c.Assert(err, chk.IsNil)

	blobURL := containerURL.NewBlockBlobURL(blobPrefix)
	blobURL.Upload(ctx, bytes.NewReader([]byte("Content")), azblob.BlobHTTPHeaders{},
		basicMetadata, azblob.BlobAccessConditions{})

	// Reference the same container URL but with anonymous credentials
	containerURL2 := azblob.NewContainerURL(containerURL.URL(), azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
	_, err = containerURL2.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	validateStorageError(c, err, azblob.ServiceCodeResourceNotFound) // Listing blobs is not publicly accessible

	// Accessing blob specific data should be public
	blobURL2 := containerURL2.NewBlockBlobURL(blobPrefix)
	resp, err := blobURL2.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestContainerCreateAccessNone(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Create(ctx, nil, azblob.PublicAccessNone)
	defer deleteContainer(c, containerURL)

	blobURL := containerURL.NewBlockBlobURL(blobPrefix)
	blobURL.Upload(ctx, bytes.NewReader([]byte("Content")), azblob.BlobHTTPHeaders{},
		basicMetadata, azblob.BlobAccessConditions{})

	// Reference the same container URL but with anonymous credentials
	containerURL2 := azblob.NewContainerURL(containerURL.URL(), azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
	// Listing blobs is not public
	_, err = containerURL2.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	validateStorageError(c, err, azblob.ServiceCodeResourceNotFound)

	// Blob data is not public
	blobURL2 := containerURL2.NewBlockBlobURL(blobPrefix)
	_, err = blobURL2.GetProperties(ctx, azblob.BlobAccessConditions{})
	c.Assert(err, chk.NotNil)
	serr := err.(azblob.StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 404) // HEAD request does not return a status code
}

func validateContainerDeleted(c *chk.C, containerURL azblob.ContainerURL) {
	_, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeContainerNotFound)
}

func (s *aztestsSuite) TestContainerDelete(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	_, err := containerURL.Delete(ctx, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	validateContainerDeleted(c, containerURL)
}

func (s *aztestsSuite) TestContainerDeleteNonExistant(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.Delete(ctx, azblob.ContainerAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeContainerNotFound)
}

func (s *aztestsSuite) TestContainerDeleteIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10) // Ensure the requests occur at different times
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	_, err := containerURL.Delete(ctx,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)
	validateContainerDeleted(c, containerURL)
}

func (s *aztestsSuite) TestContainerDeleteIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := containerURL.Delete(ctx,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestContainerDeleteIfUnModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	currentTime := getRelativeTimeGMT(10)
	_, err := containerURL.Delete(ctx,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateContainerDeleted(c, containerURL)
}

func (s *aztestsSuite) TestContainerDeleteIfUnModifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10) // Ensure the requests occur at different times

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.Delete(ctx,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestContainerAccessConditionsUnsupportedConditions(c *chk.C) {
	// This test defines that the library will panic if the user specifies conditional headers
	// that will be ignored by the service
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)

	invalidEtag := azblob.ETag("invalid")
	_, err := containerURL.SetMetadata(ctx, basicMetadata,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfMatch: invalidEtag}})
	c.Assert(err, chk.Not(chk.Equals), nil)
}

func (s *aztestsSuite) TestContainerListBlobsNonexistantPrefix(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	createNewBlockBlob(c, containerURL)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{Prefix: blobPrefix + blobPrefix})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 0)
}

func (s *aztestsSuite) TestContainerListBlobsSpecificValidPrefix(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, blobName := createNewBlockBlob(c, containerURL)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{Prefix: blobPrefix})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 1)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
}

func (s *aztestsSuite) TestContainerListBlobsValidDelimiter(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	createBlockBlobWithPrefix(c, containerURL, "a/1")
	createBlockBlobWithPrefix(c, containerURL, "a/2")
	createBlockBlobWithPrefix(c, containerURL, "b/1")
	_, blobName := createBlockBlobWithPrefix(c, containerURL, "blob")

	resp, err := containerURL.ListBlobsHierarchySegment(ctx, azblob.Marker{}, "/", azblob.ListBlobsSegmentOptions{})

	c.Assert(err, chk.IsNil)
	c.Assert(len(resp.Segment.BlobItems), chk.Equals, 1)
	c.Assert(len(resp.Segment.BlobPrefixes), chk.Equals, 2)
	c.Assert(resp.Segment.BlobPrefixes[0].Name, chk.Equals, "a/")
	c.Assert(resp.Segment.BlobPrefixes[1].Name, chk.Equals, "b/")
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
}

func (s *aztestsSuite) TestContainerListBlobsWithSnapshots(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)

	_, err := containerURL.ListBlobsHierarchySegment(ctx, azblob.Marker{}, "/", azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Snapshots: true}})
	c.Assert(err, chk.Not(chk.Equals), nil)
}

func (s *aztestsSuite) TestContainerListBlobsInvalidDelimiter(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	createBlockBlobWithPrefix(c, containerURL, "a/1")
	createBlockBlobWithPrefix(c, containerURL, "a/2")
	createBlockBlobWithPrefix(c, containerURL, "b/1")
	createBlockBlobWithPrefix(c, containerURL, "blob")

	resp, err := containerURL.ListBlobsHierarchySegment(ctx, azblob.Marker{}, "^", azblob.ListBlobsSegmentOptions{})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 4)
}

func (s *aztestsSuite) TestContainerListBlobsIncludeTypeMetadata(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, container)
	_, blobNameNoMetadata := createBlockBlobWithPrefix(c, container, "a")
	blobMetadata, blobNameMetadata := createBlockBlobWithPrefix(c, container, "b")
	_, err := blobMetadata.SetMetadata(ctx, azblob.Metadata{"field": "value"}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := container.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Metadata: true}})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobNameNoMetadata)
	c.Assert(resp.Segment.BlobItems[0].Metadata, chk.HasLen, 0)
	c.Assert(resp.Segment.BlobItems[1].Name, chk.Equals, blobNameMetadata)
	c.Assert(resp.Segment.BlobItems[1].Metadata["field"], chk.Equals, "value")
}

func (s *aztestsSuite) TestContainerListBlobsIncludeTypeSnapshots(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blob, blobName := createNewBlockBlob(c, containerURL)
	_, err := blob.CreateSnapshot(ctx, azblob.Metadata{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Snapshots: true}})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 2)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
	c.Assert(resp.Segment.BlobItems[0].Snapshot, chk.NotNil)
	c.Assert(resp.Segment.BlobItems[1].Name, chk.Equals, blobName)
	c.Assert(resp.Segment.BlobItems[1].Snapshot, chk.Equals, "")
}

func (s *aztestsSuite) TestContainerListBlobsIncludeTypeCopy(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, blobName := createNewBlockBlob(c, containerURL)
	blobCopyURL, blobCopyName := createBlockBlobWithPrefix(c, containerURL, "copy")
	_, err := blobCopyURL.StartCopyFromURL(ctx, blobURL.URL(), azblob.Metadata{}, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Copy: true}})

	// These are sufficient to show that the blob copy was in fact included
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 2)
	c.Assert(resp.Segment.BlobItems[1].Name, chk.Equals, blobName)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobCopyName)
	c.Assert(*resp.Segment.BlobItems[0].Properties.ContentLength, chk.Equals, int64(len(blockBlobDefaultData)))
	temp := blobURL.URL()
	c.Assert(*resp.Segment.BlobItems[0].Properties.CopySource, chk.Equals, temp.String())
	c.Assert(resp.Segment.BlobItems[0].Properties.CopyStatus, chk.Equals, azblob.CopyStatusSuccess)
}

func (s *aztestsSuite) TestContainerListBlobsIncludeTypeUncommitted(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, blobName := getBlockBlobURL(c, containerURL)
	_, err := blobURL.StageBlock(ctx, azblob.BlockID{0}.ToBase64(), strings.NewReader(blockBlobDefaultData), azblob.LeaseAccessConditions{}, nil)
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{UncommittedBlobs: true}})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 1)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
}

func testContainerListBlobsIncludeTypeDeletedImpl(c *chk.C, bsu azblob.ServiceURL) error {
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewBlockBlob(c, containerURL)

	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Deleted: true}})
	c.Assert(err, chk.IsNil)
	if len(resp.Segment.BlobItems) != 1 {
		return errors.New("DeletedBlobNotFound")
	}
	c.Assert(resp.Segment.BlobItems[0].Deleted, chk.Equals, true)
	return nil
}

func (s *aztestsSuite) TestContainerListBlobsIncludeTypeDeleted(c *chk.C) {
	bsu := getBSU()

	runTestRequiringServiceProperties(c, bsu, "DeletedBlobNotFound", enableSoftDelete,
		testContainerListBlobsIncludeTypeDeletedImpl, disableSoftDelete)
}

func testContainerListBlobsIncludeMultipleImpl(c *chk.C, bsu azblob.ServiceURL) error {
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)

	blobURL, blobName := createBlockBlobWithPrefix(c, containerURL, "z")
	_, err := blobURL.CreateSnapshot(ctx, azblob.Metadata{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	blobURL2, blobName2 := createBlockBlobWithPrefix(c, containerURL, "copy")
	resp2, err := blobURL2.StartCopyFromURL(ctx, blobURL.URL(), azblob.Metadata{}, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	waitForCopy(c, blobURL2, resp2)
	blobURL3, blobName3 := createBlockBlobWithPrefix(c, containerURL, "deleted")
	_, err = blobURL3.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{},
		azblob.ListBlobsSegmentOptions{Details: azblob.BlobListingDetails{Snapshots: true, Copy: true, Deleted: true}})

	c.Assert(err, chk.IsNil)
	if len(resp.Segment.BlobItems) != 5 { // If there are fewer blobs in the container than there should be, it will be because one was permanently deleted.
		return errors.New("DeletedBlobNotFound")
	}
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName2)
	c.Assert(resp.Segment.BlobItems[1].Name, chk.Equals, blobName2) // With soft delete, the overwritten blob will have a backup snapshot
	c.Assert(resp.Segment.BlobItems[2].Name, chk.Equals, blobName3)
	c.Assert(resp.Segment.BlobItems[3].Name, chk.Equals, blobName)
	c.Assert(resp.Segment.BlobItems[3].Snapshot, chk.NotNil)
	c.Assert(resp.Segment.BlobItems[4].Name, chk.Equals, blobName)
	return nil
}

func (s *aztestsSuite) TestContainerListBlobsIncludeMultiple(c *chk.C) {
	bsu := getBSU()

	runTestRequiringServiceProperties(c, bsu, "DeletedBlobNotFound", enableSoftDelete,
		testContainerListBlobsIncludeMultipleImpl, disableSoftDelete)
}

func (s *aztestsSuite) TestContainerListBlobsMaxResultsNegative(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)
	_, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{MaxResults: -2})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestContainerListBlobsMaxResultsZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	createNewBlockBlob(c, containerURL)

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{MaxResults: 0})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 1)
}

func (s *aztestsSuite) TestContainerListBlobsMaxResultsInsufficient(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, blobName := createBlockBlobWithPrefix(c, containerURL, "a")
	createBlockBlobWithPrefix(c, containerURL, "b")

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{MaxResults: 1})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 1)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
}

func (s *aztestsSuite) TestContainerListBlobsMaxResultsExact(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, blobName := createBlockBlobWithPrefix(c, containerURL, "a")
	_, blobName2 := createBlockBlobWithPrefix(c, containerURL, "b")

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{MaxResults: 2})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 2)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
	c.Assert(resp.Segment.BlobItems[1].Name, chk.Equals, blobName2)
}

func (s *aztestsSuite) TestContainerListBlobsMaxResultsSufficient(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, blobName := createBlockBlobWithPrefix(c, containerURL, "a")
	_, blobName2 := createBlockBlobWithPrefix(c, containerURL, "b")

	resp, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{MaxResults: 3})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems, chk.HasLen, 2)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)
	c.Assert(resp.Segment.BlobItems[1].Name, chk.Equals, blobName2)
}

func (s *aztestsSuite) TestContainerListBlobsNonExistentContainer(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})

	c.Assert(err, chk.NotNil)
}

func (s *aztestsSuite) TestContainerWithNewPipeline(c *chk.C) {
	bsu := getBSU()
	pipeline := testPipeline{}
	containerURL, _ := getContainerURL(c, bsu)
	containerURL = containerURL.WithPipeline(pipeline)

	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessBlob)

	c.Assert(err, chk.NotNil)
	c.Assert(err.Error(), chk.Equals, testPipelineMessage)
}

func (s *aztestsSuite) TestContainerGetSetPermissionsMultiplePolicies(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	// Define the policies
	start := generateCurrentTimeWithModerateResolution()
	expiry := start.Add(5 * time.Minute)
	expiry2 := start.Add(time.Minute)
	permissions := []azblob.SignedIdentifier{
		{ID: "0000",
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry,
				Permission: azblob.AccessPolicyPermission{Read: true, Write: true}.String(),
			},
		},
		{ID: "0001",
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry2,
				Permission: azblob.AccessPolicyPermission{Read: true}.String(),
			},
		},
	}

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, permissions,
		azblob.ContainerAccessConditions{})

	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Items, chk.DeepEquals, permissions)
}

func (s *aztestsSuite) TestContainerGetPermissionsPublicAccessNotNone(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)
	containerURL.Create(ctx, nil, azblob.PublicAccessBlob) // We create the container explicitly so we can be sure the access policy is not empty

	defer deleteContainer(c, containerURL)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.BlobPublicAccess(), chk.Equals, azblob.PublicAccessBlob)
}

func (s *aztestsSuite) TestContainerSetPermissionsPublicAccessNone(c *chk.C) {
	// Test the basic one by making an anonymous request to ensure it's actually doing it and also with GetPermissions
	// For all the others, can just use GetPermissions since we've validated that it at least registers on the server correctly
	bsu := getBSU()
	containerURL, containerName := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, blobName := createNewBlockBlob(c, containerURL)

	// Container is created with PublicAccessBlob, so setting it to None will actually test that it is changed through this method
	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	pipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	bsu2 := azblob.NewServiceURL(bsu.URL(), pipeline)
	containerURL2 := bsu2.NewContainerURL(containerName)
	blobURL2 := containerURL2.NewBlockBlobURL(blobName)
	_, err = blobURL2.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)

	// Get permissions via the original container URL so the request succeeds
	resp, _ := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})

	// If we cannot access a blob's data, we will also not be able to enumerate blobs
	validateStorageError(c, err, azblob.ServiceCodeResourceNotFound)
	c.Assert(resp.BlobPublicAccess(), chk.Equals, azblob.PublicAccessNone)
}

func (s *aztestsSuite) TestContainerSetPermissionsPublicAccessBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.BlobPublicAccess(), chk.Equals, azblob.PublicAccessBlob)
}

func (s *aztestsSuite) TestContainerSetPermissionsPublicAccessContainer(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessContainer, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.BlobPublicAccess(), chk.Equals, azblob.PublicAccessContainer)
}

func (s *aztestsSuite) TestContainerSetPermissionsACLSinglePolicy(c *chk.C) {
	bsu := getBSU()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	containerURL, containerName := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	_, blobName := createNewBlockBlob(c, containerURL)

	start := time.Now().UTC().Add(-15 * time.Second)
	expiry := start.Add(5 * time.Minute).UTC()
	permissions := []azblob.SignedIdentifier{{
		ID: "0000",
		AccessPolicy: azblob.AccessPolicy{
			Start:      start,
			Expiry:     expiry,
			Permission: azblob.AccessPolicyPermission{List: true}.String(),
		},
	}}
	_, err = containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, permissions, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	serviceSASValues := azblob.BlobSASSignatureValues{Identifier: "0000", ContainerName: containerName}
	queryParams, err := serviceSASValues.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	sasURL := bsu.URL()
	sasURL.RawQuery = queryParams.Encode()
	sasPipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	sasBlobServiceURL := azblob.NewServiceURL(sasURL, sasPipeline)

	// Verifies that the SAS can access the resource
	sasContainer := sasBlobServiceURL.NewContainerURL(containerName)
	resp, err := sasContainer.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Segment.BlobItems[0].Name, chk.Equals, blobName)

	// Verifies that successful sas access is not just because it's public
	anonymousBlobService := azblob.NewServiceURL(bsu.URL(), sasPipeline)
	anonymousContainer := anonymousBlobService.NewContainerURL(containerName)
	_, err = anonymousContainer.ListBlobsFlatSegment(ctx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	validateStorageError(c, err, azblob.ServiceCodeResourceNotFound)
}

func (s *aztestsSuite) TestContainerSetPermissionsACLMoreThanFive(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	start := time.Now().UTC()
	expiry := start.Add(5 * time.Minute).UTC()
	permissions := make([]azblob.SignedIdentifier, 6, 6)
	for i := 0; i < 6; i++ {
		permissions[i] = azblob.SignedIdentifier{
			ID: "000" + strconv.Itoa(i),
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry,
				Permission: azblob.AccessPolicyPermission{List: true}.String(),
			},
		}
	}

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, permissions, azblob.ContainerAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeInvalidXMLDocument)
}

func (s *aztestsSuite) TestContainerSetPermissionsDeleteAndModifyACL(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	start := generateCurrentTimeWithModerateResolution()
	expiry := start.Add(5 * time.Minute).UTC()
	permissions := make([]azblob.SignedIdentifier, 2, 2)
	for i := 0; i < 2; i++ {
		permissions[i] = azblob.SignedIdentifier{
			ID: "000" + strconv.Itoa(i),
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry,
				Permission: azblob.AccessPolicyPermission{List: true}.String(),
			},
		}
	}

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, permissions, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Items, chk.DeepEquals, permissions)

	permissions = resp.Items[:1] // Delete the first policy by removing it from the slice
	permissions[0].ID = "0004"   // Modify the remaining policy which is at index 0 in the new slice
	_, err = containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, permissions, azblob.ContainerAccessConditions{})

	resp, err = containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Items, chk.HasLen, 1)
	c.Assert(resp.Items, chk.DeepEquals, permissions)
}

func (s *aztestsSuite) TestContainerSetPermissionsDeleteAllPolicies(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	start := time.Now().UTC()
	expiry := start.Add(5 * time.Minute).UTC()
	permissions := make([]azblob.SignedIdentifier, 2, 2)
	for i := 0; i < 2; i++ {
		permissions[i] = azblob.SignedIdentifier{
			ID: "000" + strconv.Itoa(i),
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry,
				Permission: azblob.AccessPolicyPermission{List: true}.String(),
			},
		}
	}

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, permissions, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	_, err = containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, []azblob.SignedIdentifier{}, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Items, chk.HasLen, 0)
}

func (s *aztestsSuite) TestContainerSetPermissionsInvalidPolicyTimes(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	// Swap start and expiry
	expiry := time.Now().UTC()
	start := expiry.Add(5 * time.Minute).UTC()
	permissions := make([]azblob.SignedIdentifier, 2, 2)
	for i := 0; i < 2; i++ {
		permissions[i] = azblob.SignedIdentifier{
			ID: "000" + strconv.Itoa(i),
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry,
				Permission: azblob.AccessPolicyPermission{List: true}.String(),
			},
		}
	}

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, permissions, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestContainerSetPermissionsNilPolicySlice(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestContainerSetPermissionsSignedIdentifierTooLong(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	id := ""
	for i := 0; i < 65; i++ {
		id += "a"
	}
	expiry := time.Now().UTC()
	start := expiry.Add(5 * time.Minute).UTC()
	permissions := make([]azblob.SignedIdentifier, 2, 2)
	for i := 0; i < 2; i++ {
		permissions[i] = azblob.SignedIdentifier{
			ID: id,
			AccessPolicy: azblob.AccessPolicy{
				Start:      start,
				Expiry:     expiry,
				Permission: azblob.AccessPolicyPermission{List: true}.String(),
			},
		}
	}

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessBlob, permissions, azblob.ContainerAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeInvalidXMLDocument)
}

func (s *aztestsSuite) TestContainerSetPermissionsIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, container)

	_, err := container.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	resp, err := container.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.BlobPublicAccess(), chk.Equals, azblob.PublicAccessNone)
}

func (s *aztestsSuite) TestContainerSetPermissionsIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestContainerSetPermissionsIfUnModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetAccessPolicy(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.BlobPublicAccess(), chk.Equals, azblob.PublicAccessNone)
}

func (s *aztestsSuite) TestContainerSetPermissionsIfUnModifiedSinceFalse(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.SetAccessPolicy(ctx, azblob.PublicAccessNone, nil,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestContainerGetPropertiesAndMetadataNoMetadata(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	resp, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestContainerGetPropsAndMetaNonExistantContainer(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeContainerNotFound)
}

func (s *aztestsSuite) TestContainerSetMetadataEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)
	_, err := containerURL.Create(ctx, basicMetadata, azblob.PublicAccessBlob)

	defer deleteContainer(c, containerURL)

	_, err = containerURL.SetMetadata(ctx, azblob.Metadata{}, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (*aztestsSuite) TestContainerSetMetadataNil(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)
	_, err := containerURL.Create(ctx, basicMetadata, azblob.PublicAccessBlob)

	defer deleteContainer(c, containerURL)

	_, err = containerURL.SetMetadata(ctx, nil, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (*aztestsSuite) TestContainerSetMetadataInvalidField(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.SetMetadata(ctx, azblob.Metadata{"!nval!d Field!@#%": "value"}, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.NotNil)
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)
}

func (*aztestsSuite) TestContainerSetMetadataNonExistant(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	_, err := containerURL.SetMetadata(ctx, nil, azblob.ContainerAccessConditions{})
	validateStorageError(c, err, azblob.ServiceCodeContainerNotFound)
}

func (s *aztestsSuite) TestContainerSetMetadataIfModifiedSinceTrue(c *chk.C) {
	currentTime := getRelativeTimeGMT(-10)

	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	_, err := containerURL.SetMetadata(ctx, basicMetadata,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	resp, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})

	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)

}

func (s *aztestsSuite) TestContainerSetMetadataIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := containerURL.SetMetadata(ctx, basicMetadata,
		azblob.ContainerAccessConditions{ModifiedAccessConditions: azblob.ModifiedAccessConditions{IfModifiedSince: currentTime}})

	validateStorageError(c, err, azblob.ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestContainerNewBlobURL(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	blobURL := containerURL.NewBlobURL(blobPrefix)
	tempBlob := blobURL.URL()
	tempContainer := containerURL.URL()
	c.Assert(tempBlob.String(), chk.Equals, tempContainer.String()+"/"+blobPrefix)
	c.Assert(blobURL, chk.FitsTypeOf, azblob.BlobURL{})
}

func (s *aztestsSuite) TestContainerNewBlockBlobURL(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := getContainerURL(c, bsu)

	blobURL := containerURL.NewBlockBlobURL(blobPrefix)
	tempBlob := blobURL.URL()
	tempContainer := containerURL.URL()
	c.Assert(tempBlob.String(), chk.Equals, tempContainer.String()+"/"+blobPrefix)
	c.Assert(blobURL, chk.FitsTypeOf, azblob.BlockBlobURL{})
}
