package db

import "embed"

// Migrations contains versioned SQL shipped with the migration command.
//
//go:embed migrations/*.sql
var Migrations embed.FS
