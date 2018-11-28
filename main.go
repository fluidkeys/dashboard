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
	return 0
}

type exitCode = int

type jsonIndex struct {
	ReleaseNotesSignups []datastore.DateCount `json:"releaseNotesSignups"`
}
