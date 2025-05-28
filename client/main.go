package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

func (c *Client) CreateClient() (string, error) {
	reqBody := map[string]string{
		"name": "default-client",
	}
	respBody := struct {
		ClientID string `json:"client_id"`
	}{}

	err := c.doPost("/api/client", reqBody, &respBody)
	if err != nil {
		return "", err
	}
	return respBody.ClientID, nil
}

func (c *Client) CreateQueue(name string) (string, error) {
	reqBody := map[string]string{
		"name": name,
	}
	respBody := struct {
		QueueID string `json:"queue_id"`
	}{}

	err := c.doPost("/api/queue", reqBody, &respBody)
	if err != nil {
		return "", err
	}
	return respBody.QueueID, nil
}

func (c *Client) AppendMessage(queueID string, message int64) (string, error) {
	reqBody := map[string]interface{}{
		"queue_id": queueID,
		"message":  message,
	}
	respBody := struct {
		MessageID string `json:"message_id"`
	}{}

	err := c.doPost("/api/message/append", reqBody, &respBody)
	if err != nil {
		return "", err
	}
	return respBody.MessageID, nil
}

func (c *Client) ReadMessage(queueID, clientID string) (string, int64, error) {
	reqBody := map[string]string{
		"queue_id":  queueID,
		"client_id": clientID,
	}
	respBody := struct {
		MessageID string `json:"message_id"`
		Message   int64  `json:"message"`
	}{}

	err := c.doPost("/api/message/read", reqBody, &respBody)
	if err != nil {
		return "", 0, err
	}
	return respBody.MessageID, respBody.Message, nil
}

func (c *Client) ListQueues() ([]map[string]string, error) {
	respBody := struct {
		Queues []map[string]string `json:"queues"`
	}{}

	err := c.doGet("/api/queues", &respBody)
	if err != nil {
		return nil, err
	}
	return respBody.Queues, nil
}

func (c *Client) HealthCheck() (string, error) {
	respBody := struct {
		Status string `json:"status"`
	}{}

	err := c.doGet("/api/health", &respBody)
	if err != nil {
		return "", err
	}
	return respBody.Status, nil
}

func (c *Client) doPost(path string, body interface{}, out interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error status: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) doGet(path string, out interface{}) error {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error status: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func main() {
	c := NewClient("http://localhost:5000")

	// Health check
	status, _ := c.HealthCheck()
	fmt.Println("Health check:", status)

	// // Create client
	// clientID, _ := c.CreateClient()
	// fmt.Println("ClientID:", clientID)

	clientID := "7e34a2ac-6518-4c43-a559-61228b576c5e"

	// // Create queue
	// queueID, _ := c.CreateQueue("test-queue")
	// fmt.Println("QueueID:", queueID)

	queueID := "f0fac39d-b67f-4998-8164-3f365eaeed44"

	// // Append message
	// msgID, _ := c.AppendMessage(queueID, 51)
	// fmt.Println("Appended MessageID:", msgID)

	// Read message
	readMsgID, msgValue, err := c.ReadMessage(queueID, clientID)
	fmt.Printf("Read Message: id=%s, value=%d\n,err: %+v", readMsgID, msgValue, err)

	// // List queues
	// queues, _ := c.ListQueues()
	// fmt.Println("Queues:", queues)
}
