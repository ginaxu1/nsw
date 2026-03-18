package uploads

// FileMetadata represents the metadata of an uploaded file
type FileMetadata struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Key      string `json:"key"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}
