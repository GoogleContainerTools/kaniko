package dockerfile

import (
	"bytes"

	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/docker/docker/builder/dockerfile/parser"
)

func Parse(b []byte) ([]instructions.Stage, error) {
	p, err := parser.Parse(bytes.NewReader(b))
	stages, _, err := instructions.Parse(p.AST)
	if err != nil {
		return nil, err
	}

	return stages, err
}
