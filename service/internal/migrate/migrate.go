package migrate

import (
    "database/sql"
    "embed"
    "fmt"

    _ "github.com/lib/pq"
)

//go:embed sql/*.sql
var fs embed.FS

func Run(databaseURL string) error {
    db, err := sql.Open("postgres", databaseURL)
    if err != nil { return err }
    defer db.Close()

    // Very simple migrator: execute files lexicographically
    files, err := fs.ReadDir("sql")
    if err != nil { return err }
    for _, f := range files {
        if f.IsDir() { continue }
        b, err := fs.ReadFile("sql/" + f.Name())
        if err != nil { return err }
        if _, err := db.Exec(string(b)); err != nil {
            return fmt.Errorf("migration %s failed: %w", f.Name(), err)
        }
    }
    return nil
}

