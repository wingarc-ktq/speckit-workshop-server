package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// FileRepository is a minimal mock used by tests and future usecase wiring.
type FileRepository struct {
	SaveFn func(ctx context.Context, file domain.File, tagIDs []uuid.UUID) error
}

func (m *FileRepository) Save(ctx context.Context, file domain.File, tagIDs []uuid.UUID) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, file, tagIDs)
	}
	return nil
}
