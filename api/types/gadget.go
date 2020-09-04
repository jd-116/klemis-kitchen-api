package types

type Gadget struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Stock       int    `json:"stock"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
