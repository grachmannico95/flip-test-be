package domain

import "errors"

var (
	ErrUploadNotFound    = errors.New("upload not found")
	ErrInvalidCSVFormat  = errors.New("invalid CSV format")
	ErrProcessingFailed  = errors.New("processing failed")
	ErrDuplicateEvent    = errors.New("duplicate event")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidPageParams = errors.New("invalid page parameters")
)
