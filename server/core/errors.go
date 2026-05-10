package core

import "errors"

var (
	ErrServerNotFound = errors.New("DNS server not found in configuration")
	ErrServerExists   = errors.New("DNS server already in configuration")
)
