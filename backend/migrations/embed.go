// Package migrations ships the SQL migrations compiled into the binaries, so a
// deployed image never depends on a migrations directory on disk.
package migrations

import "embed"

// FS holds the goose migration files, applied in ascending file-name order.
// Names follow the goose convention: <version>_<description>.sql.
//
//go:embed *.sql
var FS embed.FS
