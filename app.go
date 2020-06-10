package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gorilla/mux"
)

type App struct {
	Router *mux.Router
	es     *elasticsearch.Client
}

var host = flag.String("host", os.Getenv("ELASTIC_HOST"), "The elastic host to use")
var index = flag.String("index", os.Getenv("INDEX_NAME"), "The index name")
var port = flag.String("port", os.Getenv("PORT"), "port")

func (a *App) Initialize(user, password string) {
	log.SetFlags(0)
	log.Println(*host)

	var err error
	a.es, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			*host,
		},
		Username: user,
		Password: password,
	})
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	res, err := a.es.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	defer res.Body.Close()
	log.Println(res)

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/student", a.createStudent).Methods("POST")
	a.Router.HandleFunc("/students", a.getStudents).Methods("GET")
}

func (a *App) Run() {
	log.Printf("Listen on port %s...", *port)
	log.Fatal(http.ListenAndServe(*port, a.Router))
}

func (a *App) createStudent(w http.ResponseWriter, r *http.Request) {
	req := esapi.IndexRequest{
		Index:   *index,
		Body:    r.Body,
		Refresh: "true",
	}

	res, err := req.Do(r.Context(), a.es)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer r.Body.Close()
	log.Print(res)
	var s Student

	if err := s.createStudent(res); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, s)
}

func (a *App) getStudents(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"name": r.URL.Query().Get("name"),
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}

	log.Printf("Handle getSongs query:%s", query)

	// Perform the search request.
	response, err := a.es.Search(
		a.es.Search.WithContext(r.Context()),
		a.es.Search.WithIndex(*index),
		a.es.Search.WithBody(&buf),
		a.es.Search.WithTrackTotalHits(true),
		a.es.Search.WithPretty(),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer response.Body.Close()
	log.Println(response)

	if response.IsError() {
		var e errorResponse
		if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
			log.Fatalf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			log.Fatalf("[%s] %s: %s",
				response.Status(),
				e.Error["type"],
				e.Error["reason"],
			)
		}
	}

	students, err := parseStudents(response)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	//defer r.Body.Close()

	respondWithJSON(w, http.StatusCreated, students)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
