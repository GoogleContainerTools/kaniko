package pull

import (
	"context"
	"time"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/moby/buildkit/source"
	"github.com/moby/buildkit/util/tracing"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func NewResolver(ctx context.Context, sm *session.Manager, imageStore images.Store, mode source.ResolveMode) remotes.Resolver {
	r := docker.NewResolver(docker.ResolverOptions{
		Client:      tracing.DefaultClient,
		Credentials: getCredentialsFromSession(ctx, sm),
	})

	if imageStore == nil || mode == source.ResolveModeForcePull {
		return r
	}

	return withLocalResolver{r, imageStore, mode}
}

func getCredentialsFromSession(ctx context.Context, sm *session.Manager) func(string) (string, string, error) {
	id := session.FromContext(ctx)
	if id == "" {
		return nil
	}
	return func(host string) (string, string, error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		caller, err := sm.Get(timeoutCtx, id)
		if err != nil {
			return "", "", err
		}

		return auth.CredentialsFunc(tracing.ContextWithSpanFromContext(context.TODO(), ctx), caller)(host)
	}
}

// A remotes.Resolver which checks the local image store if the real
// resolver cannot find the image, essentially falling back to a local
// image if one is present.
//
// We do not override the Fetcher or Pusher methods:
//
// - Fetcher is called by github.com/containerd/containerd/remotes/:fetch()
//   only after it has checked for the content locally, so avoid the
//   hassle of interposing a local-fetch proxy and simply pass on the
//   request.
// - Pusher wouldn't make sense to push locally, so just forward.

type withLocalResolver struct {
	remotes.Resolver
	is   images.Store
	mode source.ResolveMode
}

func (r withLocalResolver) Resolve(ctx context.Context, ref string) (string, ocispec.Descriptor, error) {
	if r.mode == source.ResolveModePreferLocal {
		if img, err := r.is.Get(ctx, ref); err == nil {
			return ref, img.Target, nil
		}
	}

	n, desc, err := r.Resolver.Resolve(ctx, ref)
	if err == nil {
		return n, desc, err
	}

	if r.mode == source.ResolveModeDefault {
		if img, err := r.is.Get(ctx, ref); err == nil {
			return ref, img.Target, nil
		}
	}

	return "", ocispec.Descriptor{}, err
}
