package s3

import (
	"context"
	"fmt"

	smithyauth "github.com/aws/smithy-go/auth"
)

type endpointAuthResolver struct {
	EndpointResolver EndpointResolverV2
}

var _ AuthSchemeResolver = (*endpointAuthResolver)(nil)

func (r *endpointAuthResolver) ResolveAuthSchemes(
	ctx context.Context, params *AuthResolverParameters,
) (
	[]*smithyauth.Option, error,
) {
	opts, err := r.resolveAuthSchemes(ctx, params)
	if err != nil {
		return nil, err
	}

	// a host of undocumented s3 operations can be done anonymously
	return append(opts, &smithyauth.Option{
		SchemeID: smithyauth.SchemeIDAnonymous,
	}), nil
}

func (r *endpointAuthResolver) resolveAuthSchemes(
	ctx context.Context, params *AuthResolverParameters,
) (
	[]*smithyauth.Option, error,
) {
	baseOpts, err := (&defaultAuthSchemeResolver{}).ResolveAuthSchemes(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get base options: %v", err)
	}

	endpt, err := r.EndpointResolver.ResolveEndpoint(ctx, *params.endpointParams)
	if err != nil {
		return nil, fmt.Errorf("resolve endpoint: %v", err)
	}

	endptOpts, ok := smithyauth.GetAuthOptions(&endpt.Properties)
	if !ok {
		return baseOpts, nil
	}

	// the list of options from the endpoint is authoritative, however, the
	// modeled options have some properties that the endpoint ones don't, so we
	// start from the latter and merge in
	for _, endptOpt := range endptOpts {
		if baseOpt := findScheme(baseOpts, endptOpt.SchemeID); baseOpt != nil {
			rebaseProps(endptOpt, baseOpt)
		}
	}

	return endptOpts, nil
}

// rebase the properties of dst, taking src as the base and overlaying those
// from dst
func rebaseProps(dst, src *smithyauth.Option) {
	iprops, sprops := src.IdentityProperties, src.SignerProperties

	iprops.SetAll(&dst.IdentityProperties)
	sprops.SetAll(&dst.SignerProperties)

	dst.IdentityProperties = iprops
	dst.SignerProperties = sprops
}

func findScheme(opts []*smithyauth.Option, schemeID string) *smithyauth.Option {
	for _, opt := range opts {
		if opt.SchemeID == schemeID {
			return opt
		}
	}
	return nil
}

func finalizeServiceEndpointAuthResolver(options *Options) {
	if _, ok := options.AuthSchemeResolver.(*defaultAuthSchemeResolver); !ok {
		return
	}

	options.AuthSchemeResolver = &endpointAuthResolver{
		EndpointResolver: options.EndpointResolverV2,
	}
}

func finalizeOperationEndpointAuthResolver(options *Options) {
	resolver, ok := options.AuthSchemeResolver.(*endpointAuthResolver)
	if !ok {
		return
	}

	if resolver.EndpointResolver == options.EndpointResolverV2 {
		return
	}

	options.AuthSchemeResolver = &endpointAuthResolver{
		EndpointResolver: options.EndpointResolverV2,
	}
}
