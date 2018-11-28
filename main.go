package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fluidkeys/dashboard/datastore"
	_ "github.com/lib/pq"
)

func main() {
	databaseUrl, present := os.LookupEnv("DATABASE_URL")

	if !present {
		panic("Missing DATABASE_URL, it should be e.g. " +
			"postgres://vagrant:password@localhost:5432/vagrant")
	}

	err := datastore.Initialize(databaseUrl)
	if err != nil {
		log.Panic(err)
	}

	http.HandleFunc("/json", handleJSONIndex)
	err = http.ListenAndServe(Port(), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
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

	out, err := json.MarshalIndent(responseData, "", "    ")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.Write(out)
}

type jsonIndex struct {
	ReleaseNotesSignups []datastore.DateCount `json:"releaseNotesSignups"`
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
