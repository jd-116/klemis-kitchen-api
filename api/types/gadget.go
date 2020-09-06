package types

type Gadget struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Important bool   `json:"important"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
