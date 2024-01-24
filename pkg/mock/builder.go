package mock

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/radiofrance/dib/pkg/types"
)

type Builder struct {
	ID string
}

func NewBuilder() *Builder {
	return &Builder{
		ID: uuid.NewString(),
	}
}

func (e *Builder) Build(opts types.ImageBuilderOpts) error {
	if err := os.MkdirAll("builds/"+e.ID, 0o755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create builds directory: %w", err)
	}

	by, err := json.MarshalIndent(opts, "", "\t")
	if err != nil {
		return err
	}

	if err := os.WriteFile("builds/"+e.ID+"/"+uuid.NewString()+".json", by, 0o600); err != nil {
		return fmt.Errorf("failed to write builds file: %w", err)
	}

	return nil
}
