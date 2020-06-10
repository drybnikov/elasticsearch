package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gorilla/mux"
)

var es *elasticsearch.Client

func main() {
	a := App{}
	a.Initialize("elastic", "aufNLvJgrYlHcF5EdCY8p2a1")
	a.Run()
}

func listen() {
	router := mux.NewRouter()

	router.HandleFunc("/students", getStudents).Methods("GET")
	router.HandleFunc("/students/{userID}", postStudents).Methods("POST")

	router.HandleFunc("/store", getStore).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func getStore(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handle getStore")
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Query().Get("name"))
}

func postStudents(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	userID := -1
	var err error
	if val, ok := pathParams["userID"]; ok {
		userID, err = strconv.Atoi(val)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "need a number"}`))
			return
		}
	}

	query := r.URL.Query()
	location := query.Get("location")

	w.Write([]byte(fmt.Sprintf(`{"userID": %d, "location": "%s" }`, userID, location)))
}

func getStudents(w http.ResponseWriter, r *http.Request) {
	var (
		m map[string]interface{}
	)
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

	log.Printf("Handle getSongs %s", query)

	// Perform the search request.
	response, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("students"),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer response.Body.Close()
	log.Printf("%s", response)

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
	if err := json.NewDecoder(response.Body).Decode(&m); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	var students []Student
	for _, hit := range m["hits"].(map[string]interface{})["hits"].([]interface{}) {
		// Parse the attributes/fields of the document
		doc := hit.(map[string]interface{})["_source"].(map[string]interface{})

		log.Printf("doc %s", doc)

		newStudent := Student{
			ID:           "sss",
			Name:         doc["name"].(string),
			Age:          int64(doc["age"].(float64)),
			AverageScore: doc["average_score"].(float64),
		}

		students = append(students, newStudent)
	}

	if err := json.NewEncoder(&buf).Encode(students); err != nil {
		log.Fatalf("Error encoding students: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, buf.String())
}
