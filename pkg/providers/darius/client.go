package darius

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cast"
)

// UserAgent version
const UserAgent = "linnix/1.0.0"

// Client struct
type Client struct {
	// App setting
	Client   *http.Client
	BaseURL  string
	LabelURL string
	Username string
	Password string
}

// NewClient make new Client
func NewClient(opts *Options) *Client {
	return &Client{
		Client:   &http.Client{},
		BaseURL:  opts.BaseURL,
		LabelURL: opts.LabelURL,
		Username: opts.Username,
		Password: opts.Password,
	}
}

// NewRequest make a client request
func (c *Client) NewRequest(method, path string, payload interface{}, values url.Values) (*http.Request, error) {
	if c.BaseURL == "" {
		return nil, errors.New("options base_url is empty")
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	if u.Path != "" {
		path = u.Path + path
	}

	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u = u.ResolveReference(rel)

	// Add custom options
	if values != nil {
		u.RawQuery = values.Encode()
	}

	var buf io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		data := url.Values{}
		data.Set("param", string(b))
		buf = strings.NewReader(data.Encode())
		log.Printf("payload send: %v", string(b))
	}

	log.Println("request: ", u.String())
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", UserAgent)

	return req, nil
}

func (c *Client) Auth() (*AuthResponse, error) {
	params := url.Values{}
	params.Set("username", c.Username)
	params.Set("password", c.Password)

	var response interface{}
	err := c.Get("/selectAuth.htm", &response, params)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	b, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	var data *AuthResponse
	err = json.Unmarshal(b, &data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return nil, err
	}

	if !cast.ToBool(data.Ack) {
		return nil, errors.New("ack fail")
	}

	log.Println("auth: ", data)
	return data, nil
}

func (c *Client) Do(r *http.Request, v interface{}) error {
	// Perform the request
	resp, err := c.Client.Do(r)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		fmt.Println("Error reading from reader:", err)
		return err
	}

	// Convert byte slice to string
	result := buf.String()
	result = strings.ReplaceAll(result, "'", `"`)

	err = json.Unmarshal([]byte(result), &v)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return err
	}

	return nil
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
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
