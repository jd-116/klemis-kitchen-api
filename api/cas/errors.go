package cas

// Error used to encode when duplicate IDs occur
// (used to provide more detailed feedback
// and to use the correct status code)
type CASValidationFailedError struct{}

func NewCASValidationFailedError() *CASValidationFailedError {
	return &CASValidationFailedError{}
}

func (e *CASValidationFailedError) Error() string {
	return "CAS validation request failed; try logging in again"
}
