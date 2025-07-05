package dto

import "mime/multipart"

type ImageReqDto struct {
	Images   []*multipart.FileHeader `json:"images"`
	UserName string                  `json:"username"`
}

type ImageListingReqDto struct {
	PageSize   int    `json:"page_size"`
	PageNumber int    `json:"page_number"`
	OrderBy    string `json:"order_by"`
}

type FileResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Thumbnail   string `json:"thumbnail"`    // Preview image URL
	DownloadURL string `json:"download_url"` // Download URL
	LikedCount  int    `json:"liked_count"`
}

type ImageLikeReqDto struct {
	ImageID string `json:"image_id"`
}
