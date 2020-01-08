package azblob_test

import (
	"bytes"
	"github.com/Azure/azure-storage-blob-go/azblob"
	chk "gopkg.in/check.v1"
	"strings"
	"time"
)

func (s *aztestsSuite) TestSnapshotSAS(c *chk.C) {
	//Generate URLs ----------------------------------------------------------------------------------------------------
	bsu := getBSU()
	containerURL, containerName := getContainerURL(c, bsu)
	blobURL, blobName := getBlockBlobURL(c, containerURL)

	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	defer containerURL.Delete(ctx, azblob.ContainerAccessConditions{})
	if err != nil {
		c.Fatal(err)
	}

	//Create file in container, download from snapshot to test. --------------------------------------------------------
	burl := containerURL.NewBlockBlobURL(blobName)
	data := "Hello world!"

	_, err = burl.Upload(ctx, strings.NewReader(data), azblob.BlobHTTPHeaders{ContentType: "text/plain"}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		c.Fatal(err)
	}

	//Create a snapshot & URL
	createSnapshot, err := burl.CreateSnapshot(ctx, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		c.Fatal(err)
	}

	//Format snapshot time
	snapTime, err := time.Parse(azblob.SnapshotTimeFormat, createSnapshot.Snapshot())
	if err != nil {
		c.Fatal(err)
	}

	//Get credentials & current time
	currentTime := time.Now().UTC()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}

	//Create SAS query
	snapSASQueryParams, err := azblob.BlobSASSignatureValues{
		StartTime:     currentTime,
		ExpiryTime:    currentTime.Add(48 * time.Hour),
		SnapshotTime:  snapTime,
		Permissions:   "racwd",
		ContainerName: containerName,
		BlobName:      blobName,
		Protocol:      azblob.SASProtocolHTTPS,
	}.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	//Attach SAS query to block blob URL
	p := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	snapParts := azblob.NewBlobURLParts(blobURL.URL())
	snapParts.SAS = snapSASQueryParams
	sburl := azblob.NewBlockBlobURL(snapParts.URL(), p)

	//Test the snapshot
	downloadResponse, err := sburl.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		c.Fatal(err)
	}

	downloadedData := &bytes.Buffer{}
	reader := downloadResponse.Body(azblob.RetryReaderOptions{})
	downloadedData.ReadFrom(reader)
	reader.Close()

	c.Assert(data, chk.Equals, downloadedData.String())

	//Try to delete snapshot -------------------------------------------------------------------------------------------
	_, err = sburl.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	if err != nil { //This shouldn't fail.
		c.Fatal(err)
	}

	//Create a normal blob and attempt to use the snapshot SAS against it (assuming failure) ---------------------------
	//If this succeeds, it means a normal SAS token was created.

	fsburl := containerURL.NewBlockBlobURL("failsnap")
	_, err = fsburl.Upload(ctx, strings.NewReader(data), azblob.BlobHTTPHeaders{ContentType: "text/plain"}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		c.Fatal(err) //should succeed to create the blob via normal auth means
	}

	fsburlparts := azblob.NewBlobURLParts(fsburl.URL())
	fsburlparts.SAS = snapSASQueryParams
	fsburl = azblob.NewBlockBlobURL(fsburlparts.URL(), p) //re-use fsburl as we don't need the sharedkey version anymore

	resp, err := fsburl.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	if err == nil {
		c.Fatal(resp) //This SHOULD fail. Otherwise we have a normal SAS token...
	}
}
