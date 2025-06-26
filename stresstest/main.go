package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	graphqlEndpoint = "http://localhost:58579/graphql"
	numClients      = 100
	requestsPerClient = 1
)

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func main() {
	log.Printf("Starting stress test with %d concurrent clients...\n", numClients)

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := range numClients {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			
			for j := range requestsPerClient {
				taskName := fmt.Sprintf("stress-task-%d-%d", clientID, j)
				token := fmt.Sprintf("stress-key-%d-%d", clientID, j)

				// create the request payload
				requestBody, err := json.Marshal(GraphQLRequest{
					Query: `
						mutation EnqueueJob($task: String!, $key: String!) {
							Enqueue(task: $task, token: $key) {
								id
								task
							}
						}
					`,
					Variables: map[string]any{
						"task": taskName,
						"key":  token,
					},
				})
				if err != nil {
					log.Printf("[Client %d] Error marshalling JSON: %v", clientID, err)
					return
				}

				// make the HTTP request
				resp, err := http.Post(graphqlEndpoint, "application/json", bytes.NewBuffer(requestBody))
				if err != nil {
					log.Printf("[Client %d] Error making request: %v", clientID, err)
					return
				}
				defer resp.Body.Close()

				// read and check the response
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("[Client %d] Error reading response body: %v", clientID, err)
					return
				}

				if resp.StatusCode != http.StatusOK {
					log.Printf("[Client %d] Received non-OK status: %s. Body: %s", clientID, resp.Status, string(bodyBytes))
				}

				if clientID < 5 {
					log.Printf("[Client %d] Successfully enqueued job. Response: %s", clientID, string(bodyBytes))
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)
	log.Printf("Stress test finished. %d jobs enqueued in %s.", numClients*requestsPerClient, duration)
	log.Printf("Average throughput: %.2f jobs/second.\n", float64(numClients*requestsPerClient)/duration.Seconds())
	log.Println("Now check the server logs and use GetAllJobStatus to verify completion.")
}