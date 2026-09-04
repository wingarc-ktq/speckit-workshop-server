package repo

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo/db"
)

func toDomainFile(row db.GetFileByIDRow, tagRows []pgtype.UUID) (domain.File, error) {
	ids := make([]uuid.UUID, 0, len(tagRows))
	for _, tagRow := range tagRows {
		ids = append(ids, uuid.UUID(tagRow.Bytes))
	}
	file := domain.File{
		ID:          uuid.UUID(row.ID.Bytes),
		Name:        row.Name,
		MIMEType:    row.MimeType,
		Description: valueOrEmpty(row.Description),
		StorageKey:  row.StorageKey,
		Size:        row.Size,
		UploadedAt:  row.UploadedAt.Time,
		TagIDs:      ids,
	}
	if file.UploadedAt.IsZero() {
		file.UploadedAt = time.Now().UTC()
	}
	return file, nil
}

func stringsContainsFold(s string, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
