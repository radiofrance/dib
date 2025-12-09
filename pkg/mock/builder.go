package mock

import (
	"context"
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
func (e *Builder) Build(_ context.Context, opts types.ImageBuilderOpts) error {
	err := os.MkdirAll(path.Join(ReportsDir, e.ID), 0o750)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create mock-reports directory: %w", err)
	}

	by, err := json.MarshalIndent(opts, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(ReportsDir, e.ID, uuid.NewString()+".json"), by, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write builds file: %w", err)
	}

	return nil
}
