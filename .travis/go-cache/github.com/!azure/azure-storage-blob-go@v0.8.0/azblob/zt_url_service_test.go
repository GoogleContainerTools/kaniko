package azblob_test

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	chk "gopkg.in/check.v1" // go get gopkg.in/check.v1
)

func (s *aztestsSuite) TestListContainers(c *chk.C) {
	sa := getBSU()
	resp, err := sa.ListContainersSegment(context.Background(), azblob.Marker{}, azblob.ListContainersSegmentOptions{Prefix: containerPrefix})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 200)
	c.Assert(resp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(resp.Version(), chk.Not(chk.Equals), "")
	c.Assert(len(resp.ContainerItems) >= 0, chk.Equals, true)
	c.Assert(resp.ServiceEndpoint, chk.NotNil)

	container, name := createNewContainer(c, sa)
	defer delContainer(c, container)

	md := azblob.Metadata{
		"foo": "foovalue",
		"bar": "barvalue",
	}
	_, err = container.SetMetadata(context.Background(), md, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	resp, err = sa.ListContainersSegment(context.Background(), azblob.Marker{}, azblob.ListContainersSegmentOptions{Detail: azblob.ListContainersDetail{Metadata: true}, Prefix: name})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContainerItems, chk.HasLen, 1)
	c.Assert(resp.ContainerItems[0].Name, chk.NotNil)
	c.Assert(resp.ContainerItems[0].Properties, chk.NotNil)
	c.Assert(resp.ContainerItems[0].Properties.LastModified, chk.NotNil)
	c.Assert(resp.ContainerItems[0].Properties.Etag, chk.NotNil)
	c.Assert(resp.ContainerItems[0].Properties.LeaseStatus, chk.Equals, azblob.LeaseStatusUnlocked)
	c.Assert(resp.ContainerItems[0].Properties.LeaseState, chk.Equals, azblob.LeaseStateAvailable)
	c.Assert(string(resp.ContainerItems[0].Properties.LeaseDuration), chk.Equals, "")
	c.Assert(string(resp.ContainerItems[0].Properties.PublicAccess), chk.Equals, string(azblob.PublicAccessNone))
	c.Assert(resp.ContainerItems[0].Metadata, chk.DeepEquals, md)
}

func (s *aztestsSuite) TestListContainersPaged(c *chk.C) {
	sa := getBSU()

	const numContainers = 4
	const maxResultsPerPage = 2
	const pagedContainersPrefix = "azblobspagedtest"

	containers := make([]azblob.ContainerURL, numContainers)
	for i := 0; i < numContainers; i++ {
		containers[i], _ = createNewContainerWithSuffix(c, sa, pagedContainersPrefix)
	}

	defer func() {
		for i := range containers {
			delContainer(c, containers[i])
		}
	}()

	marker := azblob.Marker{}
	iterations := numContainers / maxResultsPerPage

	for i := 0; i < iterations; i++ {
		resp, err := sa.ListContainersSegment(context.Background(), marker, azblob.ListContainersSegmentOptions{MaxResults: maxResultsPerPage, Prefix: containerPrefix + pagedContainersPrefix})
		c.Assert(err, chk.IsNil)
		c.Assert(resp.ContainerItems, chk.HasLen, maxResultsPerPage)

		hasMore := i < iterations-1
		c.Assert(resp.NextMarker.NotDone(), chk.Equals, hasMore)
		marker = resp.NextMarker
	}
}

func (s *aztestsSuite) TestAccountListContainersEmptyPrefix(c *chk.C) {
	bsu := getBSU()
	containerURL1, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL1)
	containerURL2, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL2)

	response, err := bsu.ListContainersSegment(ctx, azblob.Marker{}, azblob.ListContainersSegmentOptions{})

	c.Assert(err, chk.IsNil)
	c.Assert(len(response.ContainerItems) >= 2, chk.Equals, true) // The response should contain at least the two created containers. Probably many more
}

func (s *aztestsSuite) TestAccountListContainersIncludeTypeMetadata(c *chk.C) {
	bsu := getBSU()
	containerURLNoMetadata, nameNoMetadata := createNewContainerWithSuffix(c, bsu, "a")
	defer deleteContainer(c, containerURLNoMetadata)
	containerURLMetadata, nameMetadata := createNewContainerWithSuffix(c, bsu, "b")
	defer deleteContainer(c, containerURLMetadata)

	// Test on containers with and without metadata
	_, err := containerURLMetadata.SetMetadata(ctx, basicMetadata, azblob.ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	// Also validates not specifying MaxResults
	response, err := bsu.ListContainersSegment(ctx, azblob.Marker{},
		azblob.ListContainersSegmentOptions{Prefix: containerPrefix, Detail: azblob.ListContainersDetail{Metadata: true}})
	c.Assert(err, chk.IsNil)
	c.Assert(response.ContainerItems[0].Name, chk.Equals, nameNoMetadata)
	c.Assert(response.ContainerItems[0].Metadata, chk.HasLen, 0)
	c.Assert(response.ContainerItems[1].Name, chk.Equals, nameMetadata)
	c.Assert(response.ContainerItems[1].Metadata, chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestAccountListContainersMaxResultsNegative(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)
	_, err := bsu.ListContainersSegment(ctx,
		azblob.Marker{}, *(&azblob.ListContainersSegmentOptions{Prefix: containerPrefix, MaxResults: -2}))
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestAccountListContainersMaxResultsZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)

	defer deleteContainer(c, containerURL)

	// Max Results = 0 means the value will be ignored, the header not set, and the server default used
	resp, err := bsu.ListContainersSegment(ctx,
		azblob.Marker{}, *(&azblob.ListContainersSegmentOptions{Prefix: containerPrefix, MaxResults: 0}))

	c.Assert(err, chk.IsNil)
	// There could be existing container
	c.Assert(len(resp.ContainerItems) >= 1, chk.Equals, true)
}

func (s *aztestsSuite) TestAccountListContainersMaxResultsExact(c *chk.C) {
	// If this test fails, ensure there are no extra containers prefixed with go in the account. These may be left over if a test is interrupted.
	bsu := getBSU()
	containerURL1, containerName1 := createNewContainerWithSuffix(c, bsu, "a")
	defer deleteContainer(c, containerURL1)
	containerURL2, containerName2 := createNewContainerWithSuffix(c, bsu, "b")
	defer deleteContainer(c, containerURL2)

	response, err := bsu.ListContainersSegment(ctx,
		azblob.Marker{}, *(&azblob.ListContainersSegmentOptions{Prefix: containerPrefix, MaxResults: 2}))

	c.Assert(err, chk.IsNil)
	c.Assert(response.ContainerItems, chk.HasLen, 2)
	c.Assert(response.ContainerItems[0].Name, chk.Equals, containerName1)
	c.Assert(response.ContainerItems[1].Name, chk.Equals, containerName2)
}

func (s *aztestsSuite) TestAccountListContainersMaxResultsInsufficient(c *chk.C) {
	bsu := getBSU()
	containerURL1, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL1)
	containerURL2, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL2)

	response, err := bsu.ListContainersSegment(ctx, azblob.Marker{},
		*(&azblob.ListContainersSegmentOptions{Prefix: containerPrefix, MaxResults: 1}))

	c.Assert(err, chk.IsNil)
	c.Assert(len(response.ContainerItems), chk.Equals, 1)
}

func (s *aztestsSuite) TestAccountListContainersMaxResultsSufficient(c *chk.C) {
	bsu := getBSU()
	containerURL1, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL1)
	containerURL2, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL2)

	response, err := bsu.ListContainersSegment(ctx, azblob.Marker{},
		*(&azblob.ListContainersSegmentOptions{Prefix: containerPrefix, MaxResults: 3}))

	c.Assert(err, chk.IsNil)

	// This case could be instable, there could be existing containers, so the count should be >= 2
	c.Assert(len(response.ContainerItems) >= 2, chk.Equals, true)
}

func (s *aztestsSuite) TestAccountDeleteRetentionPolicy(c *chk.C) {
	bsu := getBSU()

	days := int32(5)
	_, err := bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: true, Days: &days}})
	c.Assert(err, chk.IsNil)

	// From FE, 30 seconds is guaranteed t be enough.
	time.Sleep(time.Second * 30)

	resp, err := bsu.GetProperties(ctx)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.DeleteRetentionPolicy.Enabled, chk.Equals, true)
	c.Assert(*resp.DeleteRetentionPolicy.Days, chk.Equals, int32(5))

	_, err = bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: false}})
	c.Assert(err, chk.IsNil)

	// From FE, 30 seconds is guaranteed t be enough.
	time.Sleep(time.Second * 30)

	resp, err = bsu.GetProperties(ctx)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.DeleteRetentionPolicy.Enabled, chk.Equals, false)
	c.Assert(resp.DeleteRetentionPolicy.Days, chk.IsNil)
}

func (s *aztestsSuite) TestAccountDeleteRetentionPolicyEmpty(c *chk.C) {
	bsu := getBSU()

	days := int32(5)
	_, err := bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: true, Days: &days}})
	c.Assert(err, chk.IsNil)

	// From FE, 30 seconds is guaranteed t be enough.
	time.Sleep(time.Second * 30)

	resp, err := bsu.GetProperties(ctx)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.DeleteRetentionPolicy.Enabled, chk.Equals, true)
	c.Assert(*resp.DeleteRetentionPolicy.Days, chk.Equals, int32(5))

	// Enabled should default to false and therefore disable the policy
	_, err = bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{}})
	c.Assert(err, chk.IsNil)

	// From FE, 30 seconds is guaranteed t be enough.
	time.Sleep(time.Second * 30)

	resp, err = bsu.GetProperties(ctx)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.DeleteRetentionPolicy.Enabled, chk.Equals, false)
	c.Assert(resp.DeleteRetentionPolicy.Days, chk.IsNil)
}

func (s *aztestsSuite) TestAccountDeleteRetentionPolicyNil(c *chk.C) {
	bsu := getBSU()

	days := int32(5)
	_, err := bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: true, Days: &days}})
	c.Assert(err, chk.IsNil)

	// From FE, 30 seconds is guaranteed t be enough.
	time.Sleep(time.Second * 30)

	resp, err := bsu.GetProperties(ctx)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.DeleteRetentionPolicy.Enabled, chk.Equals, true)
	c.Assert(*resp.DeleteRetentionPolicy.Days, chk.Equals, int32(5))

	_, err = bsu.SetProperties(ctx, azblob.StorageServiceProperties{})
	c.Assert(err, chk.IsNil)

	// From FE, 30 seconds is guaranteed t be enough.
	time.Sleep(time.Second * 30)

	// If an element of service properties is not passed, the service keeps the current settings.
	resp, err = bsu.GetProperties(ctx)
	c.Assert(err, chk.IsNil)
	c.Assert(resp.DeleteRetentionPolicy.Enabled, chk.Equals, true)
	c.Assert(*resp.DeleteRetentionPolicy.Days, chk.Equals, int32(5))

	// Disable for other tests
	bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: false}})
}

func (s *aztestsSuite) TestAccountDeleteRetentionPolicyDaysTooSmall(c *chk.C) {
	bsu := getBSU()

	days := int32(0) // Minimum days is 1. Validated on the client.
	_, err := bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: true, Days: &days}})
	c.Assert(strings.Contains(err.Error(), validationErrorSubstring), chk.Equals, true)
}

func (s *aztestsSuite) TestAccountDeleteRetentionPolicyDaysTooLarge(c *chk.C) {
	bsu := getBSU()

	days := int32(366) // Max days is 365. Left to the service for validation.
	_, err := bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: true, Days: &days}})
	validateStorageError(c, err, azblob.ServiceCodeInvalidXMLDocument)
}

func (s *aztestsSuite) TestAccountDeleteRetentionPolicyDaysOmitted(c *chk.C) {
	bsu := getBSU()

	// Days is required if enabled is true.
	_, err := bsu.SetProperties(ctx, azblob.StorageServiceProperties{DeleteRetentionPolicy: &azblob.RetentionPolicy{Enabled: true}})
	validateStorageError(c, err, azblob.ServiceCodeInvalidXMLDocument)
}
