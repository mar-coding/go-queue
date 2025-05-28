package internal

import "errors"

var (
	ErrNoHealthyNodes = errors.New("no healthy nodes available")
)
