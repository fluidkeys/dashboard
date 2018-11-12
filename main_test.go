package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluidkeys/teamserver/models"
)

type mockDB struct{}

func (mdb *mockDB) AllTeams() ([]*models.Team, error) {
	teams := make([]*models.Team, 0)
	teams = append(teams, &models.Team{"1", "Imagine Corporation", "e44e4317-fea1-414a-a176-2462a26b6825"})
	teams = append(teams, &models.Team{"2", "Flex Tech Inc.", "acbcd1a9-1b52-4014-9f5b-5bb6a809be3b"})
	return teams, nil
}

func TestBooksIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/books", nil)

	env := Env{db: &mockDB{}}
	http.HandlerFunc(env.teamsIndex).ServeHTTP(rec, req)

	expected :=
		`[{"id":"1","name":"Imagine Corporation","uuid":"e44e4317-fea1-414a-a176-2462a26b6825"},` +
			`{"id":"2","name":"Flex Tech Inc.","uuid":"acbcd1a9-1b52-4014-9f5b-5bb6a809be3b"}]`
	if expected != rec.Body.String() {
		t.Errorf("\n...expected = %v\n...obtained = %v", expected, rec.Body.String())
	}
}
