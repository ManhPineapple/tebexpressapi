package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"tebexpressapi/pkg/constant"
)

// UserAgent version
const UserAgent = "linnix/1.0.0"

// Client struct
type Client struct {
	// App setting
	Client       *http.Client
	BaseURL      *url.URL
	Token        string
	ResHeader    http.Header
	ErrorReponse error
}

// NewClient make new Client
func NewClient(domain, token string) *Client {
	httpClient := http.DefaultClient
	baseURL, _ := url.Parse(domain)

	c := &Client{Client: httpClient, BaseURL: baseURL, Token: token}

	return c
}

// NewRequest make a client request
func (c *Client) NewRequest(method, urlStr string, body interface{}, options url.Values, headers map[string]string) (*http.Request, error) {
	if headers == nil {
		headers = make(map[string]string, 0)
	}

	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	// Add custom options
	if options != nil {
		u.RawQuery = options.Encode()
	}

	// A bit of JSON ceremony
	var js []byte

	if body != nil {
		js, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(js))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", UserAgent)

	for key, _ := range headers {
		req.Header.Add(key, headers[key])
	}

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) error {
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	c.ResHeader = resp.Header

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		log.Printf("Request error : %v", string(body))
		c.ErrorReponse = errors.New(string(body))
		return constant.ErrInvalidRequest
	}

	if v != nil {
		err = json.Unmarshal(body, &v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) Count(path string, options url.Values, headers map[string]string) (int, error) {
	resource := struct {
		Count int `json:"count"`
	}{}
	err := c.Get(path, &resource, options, headers)
	return resource.Count, err
}

func (c *Client) CreateAndDo(method, path string, data interface{}, options url.Values, resource interface{}, headers map[string]string) error {
	req, err := c.NewRequest(method, path, data, options, headers)
	if err != nil {
		fmt.Println(err)
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
func (c *Client) Get(path string, resource interface{}, options url.Values, headers map[string]string) error {
	return c.CreateAndDo("GET", path, nil, options, resource, headers)
}

// Post performs a POST request for the given path and saves the result in the
// given resource.
func (c *Client) Post(path string, data interface{}, resource interface{}, headers map[string]string) error {
	return c.CreateAndDo("POST", path, data, nil, resource, headers)
}

// Put performs a PUT request for the given path and saves the result in the
// given resource.
func (c *Client) Put(path string, data interface{}, resource interface{}, headers map[string]string) error {
	return c.CreateAndDo("PUT", path, data, nil, resource, headers)
}

// Delete performs a DELETE request for the given path
func (c *Client) Delete(path string, headers map[string]string) error {
	return c.CreateAndDo("DELETE", path, nil, nil, nil, headers)
}
