package document

import "time"

type Document struct {
	ID               string
	OwnerID          string
	Title            string
	OriginalFilename string
	FileExt          string
	MimeType         string
	StorageKey       string
	CurrentVersionID *string
	SizeBytes        int64
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Version struct {
	ID         string
	DocumentID string
	VersionNo  int64
	StorageKey string
	SizeBytes  int64
	CreatedBy  *string
	CreatedAt  time.Time
}

type PublicDocument struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	OriginalFilename string    `json:"originalFilename"`
	FileExt          string    `json:"fileExt"`
	MimeType         string    `json:"mimeType"`
	SizeBytes        int64     `json:"sizeBytes"`
	UpdatedAt        time.Time `json:"updatedAt"`
	CreatedAt        time.Time `json:"createdAt"`
}

type PublicVersion struct {
	ID        string    `json:"id"`
	VersionNo int64     `json:"versionNo"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
}

func ToPublic(d Document) PublicDocument {
	return PublicDocument{
		ID:               d.ID,
		Title:            d.Title,
		OriginalFilename: d.OriginalFilename,
		FileExt:          d.FileExt,
		MimeType:         d.MimeType,
		SizeBytes:        d.SizeBytes,
		UpdatedAt:        d.UpdatedAt,
		CreatedAt:        d.CreatedAt,
	}
}

func VersionToPublic(v Version) PublicVersion {
	return PublicVersion{
		ID:        v.ID,
		VersionNo: v.VersionNo,
		SizeBytes: v.SizeBytes,
		CreatedAt: v.CreatedAt,
	}
}
