package s3

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/sdk"
	"github.com/aws/aws-sdk-go-v2/internal/sync/singleflight"
	"github.com/aws/smithy-go/container/private/cache"
	"github.com/aws/smithy-go/container/private/cache/lru"
)

const s3ExpressCacheCap = 100

const s3ExpressRefreshWindow = 1 * time.Minute

// The default S3Express provider uses an LRU cache with a capacity of 100.
//
// Credentials will be refreshed asynchronously when a Retrieve() call is made
// for cached credentials within an expiry window (1 minute, currently
// non-configurable).
type defaultS3ExpressCredentialsProvider struct {
	mu sync.Mutex
	sf singleflight.Group

	client        createSessionAPIClient
	credsCache    cache.Cache
	refreshWindow time.Duration
}

type createSessionAPIClient interface {
	CreateSession(context.Context, *CreateSessionInput, ...func(*Options)) (*CreateSessionOutput, error)
}

func newDefaultS3ExpressCredentialsProvider() *defaultS3ExpressCredentialsProvider {
	return &defaultS3ExpressCredentialsProvider{
		credsCache:    lru.New(s3ExpressCacheCap),
		refreshWindow: s3ExpressRefreshWindow,
	}
}

func (p *defaultS3ExpressCredentialsProvider) Retrieve(ctx context.Context, bucket string) (aws.Credentials, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	creds, ok := p.getCacheCredentials(bucket)
	if !ok || creds.Expired() {
		return p.awaitDoChanRetrieve(ctx, bucket)
	}

	if creds.Expires.Sub(sdk.NowTime()) <= p.refreshWindow {
		p.doChanRetrieve(ctx, bucket)
	}

	return *creds, nil
}

func (p *defaultS3ExpressCredentialsProvider) doChanRetrieve(ctx context.Context, bucket string) <-chan singleflight.Result {
	return p.sf.DoChan(bucket, func() (interface{}, error) {
		return p.retrieve(ctx, bucket)
	})
}

func (p *defaultS3ExpressCredentialsProvider) awaitDoChanRetrieve(ctx context.Context, bucket string) (aws.Credentials, error) {
	ch := p.doChanRetrieve(ctx, bucket)

	select {
	case r := <-ch:
		return r.Val.(aws.Credentials), r.Err
	case <-ctx.Done():
		return aws.Credentials{}, errors.New("s3express retrieve credentials canceled")
	}
}

func (p *defaultS3ExpressCredentialsProvider) retrieve(ctx context.Context, bucket string) (aws.Credentials, error) {
	resp, err := p.client.CreateSession(ctx, &CreateSessionInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return aws.Credentials{}, err
	}

	creds, err := credentialsFromResponse(resp)
	if err != nil {
		return aws.Credentials{}, err
	}

	p.putCacheCredentials(bucket, creds)
	return *creds, nil
}

func (p *defaultS3ExpressCredentialsProvider) getCacheCredentials(bucket string) (*aws.Credentials, bool) {
	if v, ok := p.credsCache.Get(bucket); ok {
		return v.(*aws.Credentials), true
	}

	return nil, false
}

func (p *defaultS3ExpressCredentialsProvider) putCacheCredentials(bucket string, creds *aws.Credentials) {
	p.credsCache.Put(bucket, creds)
}

func credentialsFromResponse(o *CreateSessionOutput) (*aws.Credentials, error) {
	if o.Credentials == nil {
		return nil, errors.New("s3express session credentials unset")
	}

	if o.Credentials.AccessKeyId == nil || o.Credentials.SecretAccessKey == nil || o.Credentials.SessionToken == nil || o.Credentials.Expiration == nil {
		return nil, errors.New("s3express session credentials missing one or more required fields")
	}

	return &aws.Credentials{
		AccessKeyID:     *o.Credentials.AccessKeyId,
		SecretAccessKey: *o.Credentials.SecretAccessKey,
		SessionToken:    *o.Credentials.SessionToken,
		CanExpire:       true,
		Expires:         *o.Credentials.Expiration,
	}, nil
}
