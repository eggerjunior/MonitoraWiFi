package httpapi

import "github.com/google/uuid"

func newUUID() uuid.UUID {
	return uuid.New()
}
