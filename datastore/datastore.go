package datastore

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	databaseUrl, present := os.LookupEnv("DATABASE_URL")

	if !present {
		panic("Missing DATABASE_URL, it should be e.g. " +
			"postgres://vagrant:password@localhost:5432/vagrant")
	}

	err := Initialize(databaseUrl)
	if err != nil {
		panic(err)
	}
}

// Initialize initialises a postgres database from the given databaseUrl
func Initialize(databaseUrl string) error {
	var err error
	db, err = sql.Open("postgres", databaseUrl)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	return nil
}

// NumberOfReleaseNotesSignupsLastNDays returns the number of signups to our
// release notes announcements list in the last 30 days
func NumberOfReleaseNotesSignupsLast30Days() ([]DateCount, error) {

	query := `SELECT (CURRENT_DATE - i) AS date,
	          COUNT(release_notes_signups.signed_up_at) AS count
	          FROM generate_series(0, 29) i
	          LEFT JOIN release_notes_signups ON date(release_notes_signups.signed_up_at) = CURRENT_DATE - i
	          GROUP BY date
	          ORDER BY date ASC;`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dateCounts := make([]DateCount, 0)

	for rows.Next() {
		nextDateCount := DateCount{}

		err = rows.Scan(&nextDateCount.Date, &nextDateCount.Count)
		if err != nil {
			return nil, err
		}
		dateCounts = append(dateCounts, nextDateCount)
	}

	return dateCounts, nil
}

func SetReleaseNoteSignupTimes(times []time.Time) error {
	fmt.Printf("Adding %d release note signups times\n", len(times))
	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = transaction.Exec("DELETE FROM release_notes_signups")
	if err != nil {
		transaction.Rollback()
		return err
	}

	for _, timestamp := range times {
		timestampString := timestamp.Format("2006-01-02T15:04:05")
		_, err := transaction.Exec("INSERT INTO release_notes_signups(signed_up_at) VALUES($1)", timestampString)
		if err != nil {
			transaction.Rollback()
			return err
		}
	}

	return transaction.Commit()
}

type JSONDate time.Time

func (t JSONDate) MarshalJSON() ([]byte, error) {
	asJson := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02"))
	return []byte(asJson), nil
}

type DateCount struct {
	Date  JSONDate `json:"date"`
	Count int      `json:"count"`
}

type releaseNotesSignup struct {
	ID         int64     `json:"id,omitempty"`
	SignedUpAt time.Time `json:"signed_up_at,omitempty"`
}
