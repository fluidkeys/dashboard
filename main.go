package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fluidkeys/dashboard/datastore"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/sheets/v4"

	"github.com/mmcdole/gofeed"
)

func main() {

	if len(os.Args) == 1 {
		os.Exit(runWebserver())
	} else if os.Args[1] == "collect" {
		os.Exit(runCollectors())
	} else if os.Args[1] == "--help" {
		os.Exit(printUsage())
	}
}

func printUsage() exitCode {
	usage := fmt.Sprintf(`
Usage:
	dashboard              run the webserver
	dashboard collect      run the data collectors
`)
	fmt.Print(usage)
	return 0
}

func runWebserver() exitCode {
	http.HandleFunc("/json", handleJSONIndex)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	err := http.ListenAndServe(Port(), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return 1
	}
	return 0
}

func handleJSONIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var err error

	responseData := jsonIndex{}

	responseData.ReleaseNotesSignups, err = datastore.NumberOfReleaseNotesSignupsLast30Days()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseData.TrialsStarted, err = datastore.NumberOfTrialsStartedLast30Days()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseData.DaysSinceLastRelease, err = datastore.DaysSinceLastReleaseAnnouncement()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseData.CallsArrangedNextWeek, err = datastore.NumberOfCallsArrangedNextWeek()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := json.MarshalIndent(responseData, "", "    ")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.Write(out)
}

// Port retrieves the port from the environment so we can run on Heroku
func Port() string {
	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "4747"
		fmt.Println("INFO: No PORT environment variable detected, defaulting to " + port)
	}
	return ":" + port
}

func runCollectors() exitCode {

	httpClient, err := getOauthClient()
	if err != nil {
		panic(err)
	}

	var errors []error

	err = syncReleaseAnnouncements(httpClient)
	if err != nil {
		errors = append(errors, err)
	}

	err = syncReleaseSignups(httpClient)
	if err != nil {
		errors = append(errors, err)
	}

	err = syncCallsArrangedFromCalendar(httpClient)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		fmt.Print("Errors encountered:\n")
		for _, err := range errors {
			fmt.Print(" * " + err.Error() + "\n")
		}
		return 1
	}

	fmt.Print("Done.\n")
	return 0
}

func getOauthClient() (*http.Client, error) {
	credentialsJson, got := os.LookupEnv("GOOGLE_API_CREDENTIALS_JSON")

	if !got {
		return nil, fmt.Errorf("Missing GOOGLE_API_CREDENTIALS_JSON environment variable")
	}

	config, err := google.ConfigFromJSON(
		[]byte(credentialsJson),
		"https://www.googleapis.com/auth/spreadsheets.readonly",
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}

	tokenJson, got := os.LookupEnv("GOOGLE_API_TOKEN_JSON")

	if !got {
		panic(fmt.Errorf("Missing GOOGLE_API_TOKEN_JSON environment variable"))
	}

	oauthToken := &oauth2.Token{}
	err = json.NewDecoder(strings.NewReader(tokenJson)).Decode(oauthToken)
	if err != nil {
		return nil, err
	}

	return config.Client(context.Background(), oauthToken), nil
}

func syncReleaseAnnouncements(client *http.Client) error {
	releaseAnnouncementTimes, err := getReleaseAnnouncementTimes(client)
	if err != nil {
		return err
	}

	return datastore.SetReleaseAnnouncementTimes(releaseAnnouncementTimes)
}

func getReleaseAnnouncementTimes(client *http.Client) ([]time.Time, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://www.fluidkeys.com/blog/feed.xml")
	if err != nil {
		return nil, err
	}

	announcementTimes := []time.Time{}
	for _, item := range feed.Items {
		if strings.Contains(item.Link, `/blog/release`) {
			timestamp, err := time.Parse("2006-01-02T15:04:05Z07:00", item.Published)

			if err != nil {
				return nil, fmt.Errorf("failed to parse date for '%s': %v", item.Title, err)
			}
			fmt.Printf("Release announcement: '%s' — %s — %s\n", item.Title, item.Link, timestamp)
			announcementTimes = append(announcementTimes, timestamp)
		}
	}

	if len(announcementTimes) == 0 {
		return nil, fmt.Errorf("got 0 release announcements, can't be right")
	}

	return announcementTimes, nil
}

func syncReleaseSignups(client *http.Client) error {
	signupTimes, err := getReleaseNoteSignupTimes(client)
	if err != nil {
		return err
	}

	return datastore.SetReleaseNoteSignupTimes(signupTimes)
}

func syncCallsArrangedFromCalendar(client *http.Client) error {
	callsArrangedTimes, err := getCallsArrangedFromCalendar(client)

	if err != nil {
		return err
	}

	return datastore.SetCallsArrangedTimes(callsArrangedTimes)
}

func getReleaseNoteSignupTimes(client *http.Client) ([]time.Time, error) {

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	spreadsheetId, got := os.LookupEnv("GOOGLE_SHEETS_RELEASE_SIGNUPS_ID")
	if !got {
		return nil, fmt.Errorf("Missing GOOGLE_SHEETS_RELEASE_SIGNUPS_ID environment variable")
	}

	readRange := "Recent signups!A2:B"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("No data found, length of resp.Values == 0")
	}

	var signupTimes []time.Time

	for _, row := range resp.Values {
		if timestampStr, ok := row[0].(string); !ok {
			return nil, fmt.Errorf("non-string cell in sheet: '%v'", row[0])
		} else {
			timefmt := "02/01/2006 15:04:05"
			timestamp, err := time.Parse(timefmt, timestampStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse timestamp "+
					"(expected format '%s'): %v", timefmt, err)
			}
			signupTimes = append(signupTimes, timestamp)
		}

	}
	return signupTimes, nil
}

func getCallsArrangedFromCalendar(client *http.Client) ([]time.Time, error) {
	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	calendarIds := []string{
		"paul@fluidkeys.com",
		"ian@fluidkeys.com",
	}

	eventIdTimeMap := make(map[string]time.Time)

	t := time.Now().Format(time.RFC3339)

	for _, calendarId := range calendarIds {
		events, err := srv.Events.List(calendarId).ShowDeleted(false).
			SingleEvents(true).TimeMin(t).MaxResults(50).OrderBy("startTime").Do()

		if err != nil {
			return nil, fmt.Errorf("failed to get upcoming events for %s: %v", calendarId, err)
		}

		if len(events.Items) == 0 {
			return nil, fmt.Errorf("no upcoming events for %s, seems unlikely", calendarId)

		}

		for _, event := range events.Items {
			// https://developers.google.com/calendar/v3/reference/events

			if eventLooksLikeCall(event) {
				fmt.Printf("calendar event looks like a call: '%s' %s\n", event.Summary, event.Start.DateTime)
				arrangedFor, err := time.Parse("2006-01-02T15:04:05Z07:00", event.Start.DateTime)
				if err != nil {
					panic(fmt.Errorf("failed to parse event.Start.Datetime '%s': %v", event.Start.DateTime, err))
				}
				eventIdTimeMap[event.Id] = arrangedFor
			}
		}
	}

	arrangedForTimes := []time.Time{}

	for _, time := range eventIdTimeMap {
		arrangedForTimes = append(arrangedForTimes, time)
	}

	return arrangedForTimes, nil
}

func eventLooksLikeCall(calendarEvent *calendar.Event) bool {
	titleLooksGood := false

	if strings.Contains(calendarEvent.Summary, " <> ") {
		titleLooksGood = true
	} else if strings.Contains(calendarEvent.Summary, " / Ian") {
		titleLooksGood = true
	} else if strings.Contains(calendarEvent.Summary, " / Paul") {
		titleLooksGood = true
	}

	hasSpecificTime := calendarEvent.Start.DateTime != ""

	return titleLooksGood && hasSpecificTime
}

type exitCode = int

type jsonIndex struct {
	CallsArrangedNextWeek      uint                  `json:"callsArrangedNextWeek"`
	DaysSinceLastRelease       uint                  `json:"daysSinceLastRelease"`
	MonthlyRecurringRevenueGBP uint                  `json:"monthlyRecurringRevenueGBP"`
	ReleaseNotesSignups        []datastore.DateCount `json:"releaseNotesSignups"`
	TrialsStarted              []datastore.DateCount `json:"trialsStarted"`
}
