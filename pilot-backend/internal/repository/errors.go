package repository

import "errors"

// ErrNotFound is returned by repository lookups that find no matching row.
var ErrNotFound = errors.New("repository: record not found")
