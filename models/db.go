package models

import (
	"database/sql"
)

type Datastore interface {
	AllTeams() ([]*Team, error)
}

type DB struct {
	*sql.DB
}

// NewDB populates the global db variable with an opened postgres database
func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}
