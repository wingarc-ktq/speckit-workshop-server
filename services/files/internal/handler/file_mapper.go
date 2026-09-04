package handler

import (
	"net/url"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

func fileInfoFromDomain(file domain.File) gen.FileInfo {
	ids := make([]openapi_types.UUID, 0, len(file.TagIDs))
	for _, id := range file.TagIDs {
		ids = append(ids, openapi_types.UUID(id))
	}
	return gen.FileInfo{
		Id:          openapi_types.UUID(file.ID),
		Name:        file.Name,
		Size:        file.Size,
		MimeType:    file.MIMEType,
		Description: stringPtr(file.Description),
		UploadedAt:  file.UploadedAt,
		DownloadUrl: (&url.URL{Path: "/files/" + file.ID.String() + "/download"}).String(),
		TagIds:      ids,
	}
}

func fileListFromDomain(files []domain.File, total int, page int, limit int) gen.FileListResponse {
	items := make([]gen.FileInfo, 0, len(files))
	for _, f := range files {
		items = append(items, fileInfoFromDomain(f))
	}
	return gen.FileListResponse{
		Files: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}
}

func toUUIDList(values []uuid.UUID) []openapi_types.UUID {
	ids := make([]openapi_types.UUID, 0, len(values))
	for _, id := range values {
		ids = append(ids, openapi_types.UUID(id))
	}
	return ids
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
