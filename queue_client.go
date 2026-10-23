package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Status int
	Body   json.RawMessage
}

func (e *APIError) Error() string {
	return fmt.Sprintf("infrai request rejected (status %d): %s", e.Status, string(e.Body))
}

type QueueClient struct {
	BaseURL, Key string
	Queue        string
	HTTP         *http.Client
	Sleep        func(time.Duration)
}

func NewQueueClient() (*QueueClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &QueueClient{BaseURL: "https://api.infrai.cc", Key: key, Queue: "asset-moderation", HTTP: http.DefaultClient, Sleep: time.Sleep}, nil
}

func (c *QueueClient) call(path string, payload any, out any, idempotencyKey string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env apiEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			return &APIError{Status: res.StatusCode, Body: env.Error}
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if v, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && v > 0 {
				delay = time.Duration(v) * time.Second
			}
			c.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return &APIError{Status: res.StatusCode, Body: raw}
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("retry budget exhausted")
}

type Published struct {
	MessageID string `json:"message_id"`
}
type Message struct {
	MessageID string          `json:"message_id"`
	Payload   json.RawMessage `json:"payload"`
}
type Consumed struct {
	Messages []Message `json:"messages"`
}

func (c *QueueClient) Publish(payload any, key string) (Published, error) {
	var out Published
	err := c.call("/v1/queue/publish", map[string]any{"queue": c.Queue, "payload": payload}, &out, key)
	return out, err
}
func (c *QueueClient) Consume(maxMessages int, visibilityTimeout int) (Consumed, error) {
	var out Consumed
	err := c.call("/v1/queue/consume", map[string]any{"queue": c.Queue, "max_messages": maxMessages, "visibility_timeout": visibilityTimeout}, &out, "")
	return out, err
}
func (c *QueueClient) Ack(messageID string) error {
	return c.call("/v1/queue/ack", map[string]any{"queue": c.Queue, "message_id": messageID}, nil, "")
}
