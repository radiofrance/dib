package mock

import (
	"github.com/radiofrance/dib/docker"
)

type Builder struct {
	Refs, Contexts []string
	CallCount      int
}

func (e *Builder) Build(opts docker.ImageBuilderOpts) error {
	e.Refs = append(e.Refs, opts.Tag)
	e.Contexts = append(e.Contexts, opts.Context)
	e.CallCount++
	return nil
}
