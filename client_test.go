package meraki

import (
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func testClient() Client {
	client, _ := NewClient("abc123", MaxRetries(0))
	gock.InterceptClient(client.HttpClient)
	return client
}

// ErrReader implements the io.Reader interface and fails on Read.
type ErrReader struct{}

// Read mocks failing io.Reader test cases.
func (r ErrReader) Read(buf []byte) (int, error) {
	return 0, errors.New("fail")
}

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	client, _ := NewClient("abc123", RequestTimeout(120))
	assert.Equal(t, client.ApiToken, "abc123")
	assert.Equal(t, client.HttpClient.Timeout, 120*time.Second)
}

// TestClientGet tests the Client::Get method.
func TestClientGet(t *testing.T) {
	defer gock.Off()
	client := testClient()
	var err error

	// Success
	gock.New(client.BaseUrl).Get("/url").Reply(200)
	_, err = client.Get("/url")
	assert.NoError(t, err)

	// HTTP error
	gock.New(client.BaseUrl).Get("/url").ReplyError(errors.New("fail"))
	_, err = client.Get("/url")
	assert.Error(t, err)

	// Invalid HTTP status code
	gock.New(client.BaseUrl).Get("/url").Reply(405)
	_, err = client.Get("/url")
	assert.Error(t, err)

	// Error decoding response body
	gock.New(client.BaseUrl).
		Get("/url").
		Reply(200).
		Map(func(res *http.Response) *http.Response {
			res.Body = io.NopCloser(ErrReader{})
			return res
		})
	_, err = client.Get("/url")
	assert.Error(t, err)
}

// TestClientDelete tests the Client::Delete method.
func TestClientDelete(t *testing.T) {
	defer gock.Off()
	client := testClient()

	// Success
	gock.New(client.BaseUrl).
		Delete("/url").
		Reply(200)
	_, err := client.Delete("/url")
	assert.NoError(t, err)

	// HTTP error
	gock.New(client.BaseUrl).
		Delete("/url").
		ReplyError(errors.New("fail"))
	_, err = client.Delete("/url")
	assert.Error(t, err)
}

// TestClientPost tests the Client::Post method.
func TestClientPost(t *testing.T) {
	defer gock.Off()
	client := testClient()

	var err error

	// Success
	gock.New(client.BaseUrl).Post("/url").Reply(200)
	_, err = client.Post("/url", "{}")
	assert.NoError(t, err)

	// HTTP error
	gock.New(client.BaseUrl).Post("/url").ReplyError(errors.New("fail"))
	_, err = client.Post("/url", "{}")
	assert.Error(t, err)

	// Invalid HTTP status code
	gock.New(client.BaseUrl).Post("/url").Reply(405)
	_, err = client.Post("/url", "{}")
	assert.Error(t, err)

	// Error decoding response body
	gock.New(client.BaseUrl).
		Post("/url").
		Reply(200).
		Map(func(res *http.Response) *http.Response {
			res.Body = io.NopCloser(ErrReader{})
			return res
		})
	_, err = client.Post("/url", "{}")
	assert.Error(t, err)
}

// TestClientPost tests the Client::Post method.
func TestClientPut(t *testing.T) {
	defer gock.Off()
	client := testClient()

	var err error

	// Success
	gock.New(client.BaseUrl).Put("/url").Reply(200)
	_, err = client.Put("/url", "{}")
	assert.NoError(t, err)

	// HTTP error
	gock.New(client.BaseUrl).Put("/url").ReplyError(errors.New("fail"))
	_, err = client.Put("/url", "{}")
	assert.Error(t, err)

	// Invalid HTTP status code
	gock.New(client.BaseUrl).Put("/url").Reply(405)
	_, err = client.Put("/url", "{}")
	assert.Error(t, err)

	// Error decoding response body
	gock.New(client.BaseUrl).
		Put("/url").
		Reply(200).
		Map(func(res *http.Response) *http.Response {
			res.Body = io.NopCloser(ErrReader{})
			return res
		})
	_, err = client.Put("/url", "{}")
	assert.Error(t, err)
}
