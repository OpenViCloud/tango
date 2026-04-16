package whatsapp

import (
	"database/sql"

	_ "github.com/glebarez/go-sqlite"
)

func init() {
	// whatsmeow's sqlstore uses driver name "sqlite3".
	// modernc.org/sqlite registers as "sqlite" (Pure Go, no CGO).
	// Re-register the same driver under "sqlite3" for compatibility.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic("sqlite_compat: failed to open sqlite driver: " + err.Error())
	}
	drv := db.Driver()
	db.Close()
	sql.Register("sqlite3", drv)
}
