package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gorilla/mux"
)

type App struct {
	Router *mux.Router
	es     *elasticsearch.Client
}

func (a *App) Initialize(user, password string) {
	log.SetFlags(0)

	var err error
	a.es, err = elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/student", a.createStudent).Methods("POST")
}

func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, a.Router))
}

func (a *App) createStudent(w http.ResponseWriter, r *http.Request) {
	req := esapi.IndexRequest{
		Index: "students",
		Body:  r.Body,
		//DocumentID: r.URL.Query().Get("id"),
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

	if err := s.createStudent(*res); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, s)
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
