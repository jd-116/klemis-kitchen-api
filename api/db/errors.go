package db

import "fmt"

// Error used to encode when duplicate IDs occur
// (used to provide more detailed feedback
// and to use the correct status code)
type DuplicateIDError struct {
	OriginalID string
}

func NewDuplicateIDError(originalID string) *DuplicateIDError {
	return &DuplicateIDError{
		OriginalID: originalID,
	}
}

func (e *DuplicateIDError) Error() string {
	return fmt.Sprintf("given ID '%s' collides with existing IDs in the database",
		e.OriginalID)
}

// Error used to encode when an ID isn't found
// for GetSingle, Update, and Delete operations
type NotFoundError struct {
	ID string
}

func NewNotFoundError(id string) *NotFoundError {
	return &NotFoundError{
		ID: id,
	}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("object with ID '%s' not found in the database",
		e.ID)
}
