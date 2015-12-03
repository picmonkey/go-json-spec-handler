package jsh

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Request is just a wrapper around an http.Request to make sending more fluent
type Request struct {
	*http.Request
}

// ClientResponse is a wrapper around an http.Response that allows us to perform
// intelligent actions on them
type ClientResponse struct {
	*http.Response
}

// GetObject validates the http response and parses out the JSON object from the
// body if possible
func (c *ClientResponse) GetObject() (*Object, SendableError) {
	return parseSingle(c.Header, c.Body)
}

// GetList validates the http response and parses out the JSON list from the
// body if possible
func (c *ClientResponse) GetList() ([]*Object, SendableError) {
	return parseMany(c.Header, c.Body)
}

// Send sends an http.Request and handles parsing the response back
func (r *Request) Send() (*ClientResponse, error) {
	client := &http.Client{}

	res, err := client.Do(r.Request)
	if err != nil {
		return nil, err
	}

	return &ClientResponse{res}, nil
}

// NewGetRequest allows a user to make an outbound GET /resource(/:id) request.
//
// For a GET request that retrieves multiple resources, pass an empty string for
// the id parameter:
//
//	request, err := jsh.NewGetRequest("http://apiserver", "user", "")
//	resp, err := request.Send() // GET "http://apiserver/users
//
// For a GET request on a specific attribute:
//
//	request, err := jsh.NewGetRequest("http://apiserver", "user", "2")
//	resp, err := request.Send() // GET "http://apiserver/users/2
//
func NewGetRequest(urlStr string, resourceType string, id string) (*Request, error) {

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// ghetto pluralization, fix when it becomes an issue
	setPath(u, resourceType, id)

	request, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating new HTTP request: %s", err.Error())
	}

	return &Request{request}, nil
}

// NewRequest creates a new JSON Spec compatible http.Request for
// PATCH, DELETE, and POST http.Request method types. Use NewGetRequest for a GET
// based http.Request.
//
// Example:
//
//  obj, err := jsh.NewObject("123", "objtype", payload)
//  req, err := jsh.NewRequest("POST", "http://postap.com", obj)
//  resp, err := req.Send()
//
func NewRequest(method string, urlStr string, object *Object) (*Request, error) {

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	switch method {
	case "GET":
		return nil, ISE("Use jsh.NewGetRequest() for 'GET' method http requests")
	case "PATCH":
	case "DELETE":
		if object == nil {
			return nil, SpecificationError(fmt.Sprintf(
				"Object must be present for HTTP method '%s'", method,
			))
		}
		setPath(u, object.Type, object.ID)
		break
	case "POST":
		setPath(u, object.Type, object.ID)
		break
	default:
		return nil, SpecificationError(fmt.Sprintf(
			"Cannot use HTTP method '%s' for a JSON Request", method,
		))
	}

	request, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating new HTTP request: %s", err.Error())
	}

	var content []byte

	// use Prepare to generate a payload
	if object != nil {

		payload, err := object.prepare(request, false)
		if err != nil {
			return nil, fmt.Errorf("Error preparing object: %s", err.Error())
		}

		jsonContent, jsonErr := json.MarshalIndent(payload, "", "  ")
		if jsonErr != nil {
			return nil, fmt.Errorf("Unable to prepare JSON content: %s", jsonErr)
		}

		content = jsonContent
	}

	request.Body = CreateReadCloser(content)
	request.Header.Add("Content-Type", ContentType)
	request.Header.Set("Content-Length", strconv.Itoa(len(content)))

	return &Request{request}, nil
}

func setPath(url *url.URL, resource string, id string) {

	if url.Path != "" && !strings.HasSuffix(url.Path, "/") {
		url.Path = url.Path + "/"
	}

	url.Path = fmt.Sprintf("%s%ss", url.Path, resource)

	if id != "" {
		url.Path = strings.Join([]string{url.Path, id}, "/")
	}
}
