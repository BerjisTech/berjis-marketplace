package db

import (
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

func Connect(url string) (*sqlx.DB, error) {
    return sqlx.Connect("postgres", url)
}

