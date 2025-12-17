package dto

type CreatePostRequest struct {
	Content string   `json:"content" validate:"required,max=1000"`
	Title   string   `json:"title" validate:"required,max=100"`
	Tags    []string `json:"tags"`
}

type UpdatePostRequest struct {
	Content *string `json:"content" validate:"omitempty,max=1000"` // nullable
	Title   *string `json:"title" validate:"omitempty,max=100"`    // nullable
}
