package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type errorResponse struct {
	Error map[string]interface{}
}

type Student struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Age          int64   `json:"age"`
	AverageScore float64 `json:"average_score"`
}

func (s *Student) deleteStudent(es *elasticsearch.Client) error {
	req := esapi.DeleteRequest{
		Index:      os.Getenv("INDEX_NAME"),
		DocumentID: s.ID,
	}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	defer res.Body.Close()
	log.Print(res)

	return err
}

func (s *Student) createStudent(res *esapi.Response) error {
	var req map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&req)

	if err != nil {
		log.Printf("Error parsing the response body: %s", err)
	} else {
		log.Printf("[%s] %s; version=%d", res.Header, req["result"], int(req["_version"].(float64)))
		s.ID = req["_id"].(string)
	}

	defer res.Body.Close()

	return err
}

func parseStudents(res *esapi.Response) (*[]Student, error) {
	var students []Student
	var m map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
		return nil, err
	}
	for _, hit := range m["hits"].(map[string]interface{})["hits"].([]interface{}) {
		// Parse the attributes/fields of the document
		id := hit.(map[string]interface{})["_id"].(string)
		doc := hit.(map[string]interface{})["_source"].(map[string]interface{})

		log.Printf("doc %s", doc)

		newStudent := Student{
			ID:           id,
			Name:         doc["name"].(string),
			Age:          int64(doc["age"].(float64)),
			AverageScore: doc["average_score"].(float64),
		}

		students = append(students, newStudent)
	}

	return &students, nil
}
