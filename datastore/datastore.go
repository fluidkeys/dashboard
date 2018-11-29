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
	return getRowCountLast30Days("release_notes_signups", "signed_up_at")
}

// NumberOfTrialsStartedLast30Days returns 30 entries (oldest to newest) of
// the number of team trials started on each date
func NumberOfTrialsStartedLast30Days() ([]DateCount, error) {
	return getRowCountLast30Days("trials_started", "started_at")
}

func getRowCountLast30Days(tableName string, columnName string) ([]DateCount, error) {

	query := fmt.Sprintf(`SELECT (CURRENT_DATE - i) AS date,
	          COUNT(%s.%s) AS count
	          FROM generate_series(0, 29) i
	          LEFT JOIN %s ON date(%s.%s) = CURRENT_DATE - i
	          GROUP BY date
	          ORDER BY date ASC;`, tableName, columnName, tableName, tableName, columnName)

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

func NumberOfCallsArrangedNext7Days() (uint, error) {
	query := `SELECT COUNT(arranged_for) AS count
	          FROM calls_arranged
		  WHERE calls_arranged.arranged_for > now()
		  AND calls_arranged.arranged_for <= now() + interval '1 week'`

	var count uint
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func DaysSinceLastReleaseAnnouncement() (uint, error) {
	query := `SELECT date_part('day', NOW() - published_at) AS days_ago
	          FROM release_announcements
		  ORDER BY published_at DESC
		  LIMIT 1;`

	var daysAgo uint
	err := db.QueryRow(query).Scan(&daysAgo)
	if err != nil {
		return 0, err
	}
	return daysAgo, nil
}

func SetReleaseNoteSignupTimes(times []time.Time) error {
	fmt.Printf("Adding %d release note signups times\n", len(times))

	return replaceTimeRowsWith(times, "release_notes_signups", "signed_up_at")
}

func SetCallsArrangedTimes(times []time.Time) error {
	fmt.Printf("Adding %d calls arranged times\n", len(times))

	return replaceTimeRowsWith(times, "calls_arranged", "arranged_for")
}

func SetReleaseAnnouncementTimes(times []time.Time) error {
	fmt.Printf("Adding %d release announcement times\n", len(times))

	return replaceTimeRowsWith(times, "release_announcements", "published_at")
}

// replaceTimeRowsWith deletes *all rows* in the given tableName then re-inserts
// the given `times` into `columnName`
// This is done in a transaction so a failure will rollback to the original state
func replaceTimeRowsWith(times []time.Time, tableName string, columnName string) error {
	fmt.Printf("tableName: '%s'\n", tableName)
	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = transaction.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	if err != nil {
		transaction.Rollback()
		return err
	}

	for _, timestamp := range times {
		timestampString := timestamp.Format("2006-01-02T15:04:05")
		query := fmt.Sprintf("INSERT INTO %s(%s) VALUES($1)", tableName, columnName)

		_, err := transaction.Exec(query, timestampString)
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
