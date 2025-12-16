package handler

import "errors"

var (
	// ErrFailedToParseCABundle is returned when the serviceaccount CA bundle cannot be parsed.
	ErrFailedToParseCABundle = errors.New("failed to parse serviceaccount CA bundle")
)
