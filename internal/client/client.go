package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Client wraps Cockpit API interaction.

type Client struct {
	BaseURL *url.URL
	Token   string
	*http.Client
}

func New(rawURL, token string) (*Client, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &Client{BaseURL: u, Token: token, Client: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) apiRequest(path string, body any, v any) error {
	// Compose URL with token query param
	q := url.Values{}
	q.Set("token", c.Token)

	full := c.BaseURL.ResolveReference(&url.URL{Path: path, RawQuery: q.Encode()})

	var req *http.Request
	var err error
	if body != nil {
		buf, _ := json.Marshal(body)
		req, err = http.NewRequest(http.MethodPost, full.String(), bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(http.MethodGet, full.String(), nil)
	}
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	if v == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// ListCollections returns all collection names.
func (c *Client) ListCollections() ([]string, error) {
	var out []string
	if err := c.apiRequest("/api/collections/listCollections", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchDocuments returns raw entries of a collection (no filter, limited by API default (or 0)).
func (c *Client) FetchDocuments(coll string) ([]json.RawMessage, error) {
	// Empty payload yields all docs (subject to default limit 0 = unlimited in Cockpit)
	payload := map[string]any{}
	var res struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := c.apiRequest("/api/collections/get/"+coll, payload, &res); err != nil {
		return nil, err
	}
	return res.Entries, nil
}

// Document represents a collection entry plus metadata.
type Document struct {
	ID       string          `json:"_id"`
	Rev      int64           `json:"_modified"`
	Raw      json.RawMessage `json:"-"`
	Filename string          `json:"-"`
}

// GetDoc downloads an entry and persists it under docs/.
func (c *Client) GetDoc(coll, id string) (*Document, error) {
	payload := map[string]any{"filter": map[string]string{"_id": id}, "limit": 1}
	var res struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := c.apiRequest("/api/collections/get/"+coll, payload, &res); err != nil {
		return nil, err
	}
	if len(res.Entries) == 0 {
		return nil, errors.New("document not found")
	}
	raw := res.Entries[0]
	var meta struct {
		ID  string `json:"_id"`
		Rev int64  `json:"_modified"`
	}
	_ = json.Unmarshal(raw, &meta)

	// Persist to disk for editing
	if err := os.MkdirAll("docs", 0o755); err != nil {
		return nil, err
	}
	filename := filepath.Join("docs", coll+"-"+meta.ID+".json")
	if err := os.WriteFile(filename, raw, 0o644); err != nil {
		return nil, err
	}

	return &Document{ID: meta.ID, Rev: meta.Rev, Raw: raw, Filename: filename}, nil
}

// UpdateDoc uploads a raw JSON document to its collection and returns new revision.
func (c *Client) UpdateDoc(coll string, docRaw []byte) (int64, error) {
	var meta struct {
		ID  string `json:"_id"`
		Rev int64  `json:"_modified"`
	}
	if err := json.Unmarshal(docRaw, &meta); err != nil {
		return 0, err
	}
	if meta.ID == "" {
		return 0, errors.New("document JSON missing _id field")
	}

	payload := map[string]any{"data": json.RawMessage(docRaw)}
	var res struct {
		Data struct {
			Rev int64 `json:"_modified"`
		} `json:"data"`
	}
	if err := c.apiRequest("/api/collections/save/"+coll, payload, &res); err != nil {
		return 0, err
	}
	return res.Data.Rev, nil
}
