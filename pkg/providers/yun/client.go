package yun

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
)

// UserAgent version
const UserAgent = "linnix/1.0.0"

// Client struct
type Client struct {
	// App setting
	Client   *http.Client
	BaseURL  string
	UserName string `json:"username"`
	Password string `json:"password"`
}

// NewClient make new Client
func NewClient(domain string, userName, password string) *Client {
	newClient := &Client{
		Client:   &http.Client{},
		BaseURL:  domain,
		UserName: userName,
		Password: password,
	}

	return newClient
}

// NewRequest make a client request
func (c *Client) NewRequest(method, path string, payload interface{}, options url.Values) (*http.Request, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u = u.ResolveReference(rel)

	// Add custom options
	if options != nil {
		u.RawQuery = options.Encode()
	}

	var buf io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		buf = bytes.NewBuffer(b)
	}

	//log.Println("request: ", u.String())
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Authorization", "Basic "+basicAuth(c.UserName, c.Password))

	return req, nil
}

func (c *Client) Do(r *http.Request, v interface{}) error {
	res, err := c.Client.Do(r)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing: %v", err)
		}
	}(res.Body)

	if v == nil {
		return nil
	}

	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(res.Body).Decode(v)
}

func (c *Client) CreateAndDo(method, path string, data interface{}, options url.Values, resource interface{}) error {
	req, err := c.NewRequest(method, path, data, options)
	if err != nil {
		return err
	}

	err = c.Do(req, resource)
	if err != nil {
		return err
	}

	return nil
}

// Get performs a GET request for the given path and saves the result in the
// given resource.
func (c *Client) Get(path string, resource interface{}, options url.Values) error {
	return c.CreateAndDo("GET", path, nil, options, resource)
}

// Post performs a POST request for the given path and saves the result in the
// given resource.
func (c *Client) Post(path string, data interface{}, resource interface{}) error {
	return c.CreateAndDo("POST", path, data, nil, resource)
}

// PUT performs a PUT request for the given path and saves the result in the
// given resource.
func (c *Client) Put(path string, data interface{}, resource interface{}) error {
	return c.CreateAndDo("PUT", path, data, nil, resource)
}

// Delete performs a delete request for the given path and saves the result in the
// given resource
func (c *Client) Delete(path string, data interface{}, resource interface{}) error {
	return c.CreateAndDo("DELETE", path, data, nil, resource)
}

func basicAuth(username, password string) string {
	auth := username + "&" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
