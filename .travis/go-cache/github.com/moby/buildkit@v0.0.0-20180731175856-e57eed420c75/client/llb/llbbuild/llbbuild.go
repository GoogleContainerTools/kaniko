package llbbuild

import (
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/apicaps"
	digest "github.com/opencontainers/go-digest"
)

func Build(opt ...BuildOption) llb.StateOption {
	return func(s llb.State) llb.State {
		return s.WithOutput(NewBuildOp(s.Output(), opt...).Output())
	}
}

func NewBuildOp(source llb.Output, opt ...BuildOption) llb.Vertex {
	info := &BuildInfo{}
	for _, o := range opt {
		o(info)
	}
	return &build{source: source, info: info, constraints: info.Constraints}
}

type build struct {
	llb.MarshalCache
	source         llb.Output
	info           *BuildInfo
	cachedPBDigest digest.Digest
	cachedPB       []byte
	constraints    llb.Constraints
}

func (b *build) ToInput(c *llb.Constraints) (*pb.Input, error) {
	dgst, _, _, err := b.Marshal(c)
	if err != nil {
		return nil, err
	}
	return &pb.Input{Digest: dgst, Index: pb.OutputIndex(0)}, nil
}

func (b *build) Vertex() llb.Vertex {
	return b
}

func (b *build) Validate() error {
	return nil
}

func (b *build) Marshal(c *llb.Constraints) (digest.Digest, []byte, *pb.OpMetadata, error) {
	if b.Cached(c) {
		return b.Load()
	}
	pbo := &pb.BuildOp{
		Builder: pb.LLBBuilder,
		Inputs: map[string]*pb.BuildInput{
			pb.LLBDefinitionInput: {pb.InputIndex(0)}},
	}

	pbo.Attrs = map[string]string{}

	if b.info.DefinitionFilename != "" {
		pbo.Attrs[pb.AttrLLBDefinitionFilename] = b.info.DefinitionFilename
	}

	if b.constraints.Metadata.Caps == nil {
		b.constraints.Metadata.Caps = make(map[apicaps.CapID]bool)
	}
	b.constraints.Metadata.Caps[pb.CapBuildOpLLBFileName] = true

	pop, md := llb.MarshalConstraints(c, &b.constraints)
	pop.Op = &pb.Op_Build{
		Build: pbo,
	}

	inp, err := b.source.ToInput(c)
	if err != nil {
		return "", nil, nil, err
	}

	pop.Inputs = append(pop.Inputs, inp)

	dt, err := pop.Marshal()
	if err != nil {
		return "", nil, nil, err
	}
	b.Store(dt, md, c)
	return b.Load()
}

func (b *build) Output() llb.Output {
	return b
}

func (b *build) Inputs() []llb.Output {
	return []llb.Output{b.source}
}

type BuildInfo struct {
	llb.Constraints
	DefinitionFilename string
}

type BuildOption func(*BuildInfo)

func WithFilename(fn string) BuildOption {
	return func(b *BuildInfo) {
		b.DefinitionFilename = fn
	}
}

func WithConstraints(co llb.ConstraintsOpt) BuildOption {
	return func(b *BuildInfo) {
		co.SetConstraintsOption(&b.Constraints)
	}
}
