package models

// A Team represents a Fluidkeys team that use the server
type Team struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	UUID string `json:"uuid,omitempty"`
}

// AllTeams reads all the teams in the database
func (db *DB) AllTeams() ([]*Team, error) {
	teams := make([]*Team, 0)
	rows, err := db.Query(`
		SELECT id, name, uuid FROM teams`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		team := Team{}
		err = rows.Scan(&team.ID, &team.Name, &team.UUID)
		if err != nil {
			return nil, err
		}
		teams = append(teams, &team)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return teams, nil
}
