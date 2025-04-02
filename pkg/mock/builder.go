package mock

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

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

const ReportsDir = "tests/mock-reports"

//nolint:musttag
func (e *Builder) Build(opts types.ImageBuilderOpts) error {
	if err := os.MkdirAll(path.Join(ReportsDir, e.ID), 0o750); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create mock-reports directory: %w", err)
	}

	by, err := json.MarshalIndent(opts, "", "\t")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(ReportsDir, e.ID, uuid.NewString()+".json"), by, 0o600); err != nil {
		return fmt.Errorf("failed to write builds file: %w", err)
	}

	return nil
}
