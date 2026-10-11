package dto

type SuccessResponse[TResponse any] struct {
	Status   string    `json:"status"`
	Data     TResponse `json:"data"`
	Message  any       `json:"message"`
	MetaData any       `json:"meta_data"`
}

// ErrorResponse keeps the starter's envelope and adds a stable machine-readable
// code. Errors carries the error details (for example available tickets).
type ErrorResponse struct {
	Status   string `json:"status"`
	Code     string `json:"code"`
	Errors   any    `json:"errors"`
	Message  any    `json:"message"`
	MetaData any    `json:"meta_data"`
}
