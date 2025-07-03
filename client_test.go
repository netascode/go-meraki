package meraki

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/sjson"
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

// TestClientGetPages is like TestClientGet, but with basic pagination.
func TestClientGetPages(t *testing.T) {
	defer gock.Off()
	client := testClient()
	var err error

	gock.New(client.BaseUrl).Get("/url").
		Reply(200).
		BodyString(`["1","2","3"]`).
		Header.Set("Link", `<`+client.BaseUrl+`/url?offset=4>; rel="next"`)
	gock.New(client.BaseUrl).Get("/url").MatchParam("offset", "4").
		Reply(200).
		BodyString(`["4","5","6"]`).
		Header.Set("Link", `<`+client.BaseUrl+`/url?offset=7>; rel="next"`)
	gock.New(client.BaseUrl).Get("/url").MatchParam("offset", "7").
		Reply(200).
		BodyString(`["7","8"]`).
		Header.Set("Link", `<`+client.BaseUrl+`/url?offset=1>; rel="first"`)

	res, err := client.Get("/url")
	assert.NoError(t, err)
	assert.Equal(t, `["1","2","3","4","5","6","7","8"]`, res.Raw)
}

// TestClientGetPages is like TestClientGet, but with basic pagination and "items" response.
func TestClientGetPagesItems(t *testing.T) {
	defer gock.Off()
	client := testClient()
	var err error

	gock.New(client.BaseUrl).Get("/url").
		Reply(200).
		BodyString(`{"items": ["1","2","3"]}`).
		Header.Set("Link", `<`+client.BaseUrl+`/url?offset=4>; rel="next"`)
	gock.New(client.BaseUrl).Get("/url").MatchParam("offset", "4").
		Reply(200).
		BodyString(`{"items": ["4","5","6"]}`).
		Header.Set("Link", `<`+client.BaseUrl+`/url?offset=7>; rel=next`)
	gock.New(client.BaseUrl).Get("/url").MatchParam("offset", "7").
		Reply(200).
		BodyString(`{"items": ["7","8"]}`).
		Header.Set("Link", `<`+client.BaseUrl+`/url?offset=1>; rel="first"`)

	res, err := client.Get("/url")
	assert.NoError(t, err)
	assert.Equal(t, `{"items":["1","2","3","4","5","6","7","8"]}`, res.Raw)
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

// TestClientBatch tests the Client::Batch method (all scenarios)
func TestClientBatch(t *testing.T) {
	defer gock.Off()
	client := testClient()

	// Synchronous Success
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		Reply(201).
		BodyString(`{"id": "1", "status": {"completed": true, "failed": false, "errors": []}}`)
	actions := []ActionModel{NewAction("update", "url", "{}")}
	_, err := client.Batch("1", actions)
	assert.NoError(t, err)

	// Asynchronous Success
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		Reply(201).
		BodyString(`{"id": "1", "status": {"completed": false, "failed": false, "errors": []}}`)
	gock.New(client.BaseUrl).
		Get("/organizations/1/actionBatches/1").
		Reply(200).
		BodyString(`{"id": "1", "status": {"completed": true, "failed": false, "errors": []}}`)
	actions = make([]ActionModel, 21)
	for i := range actions {
		actions[i] = NewAction("update", fmt.Sprintf("url%d", i), "{}")
	}
	_, err = client.Batch("1", actions)
	assert.NoError(t, err)

	// Error
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		Reply(201).
		BodyString(`{"id": "1", "status": {"completed": false, "failed": true, "errors": ["Unsupported operation"]}}`)
	actions = []ActionModel{NewAction("update", "url", "{}")}
	_, err = client.Batch("1", actions)
	assert.Error(t, err)

	// Multi-batch Success (2 batches: 100 + 10)
	actions = make([]ActionModel, 110)
	for i := range actions {
		actions[i] = NewAction("create", fmt.Sprintf("url%d", i), "{}")
	}
	createdResources1 := "[]"
	for i := range 100 {
		item, _ := sjson.Set("", "id", fmt.Sprintf("%d", i))
		item, _ = sjson.Set(item, "url", fmt.Sprintf("url%d", i))
		createdResources1, _ = sjson.SetRaw(createdResources1, "-1", item)
	}
	createdResources2 := "[]"
	for i := 100; i < 110; i++ {
		item, _ := sjson.Set("", "id", fmt.Sprintf("%d", i))
		item, _ = sjson.Set(item, "url", fmt.Sprintf("url%d", i))
		createdResources2, _ = sjson.SetRaw(createdResources2, "-1", item)
	}
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		Reply(201).
		BodyString(fmt.Sprintf(`{"id": "1", "status": {"completed": true, "failed": false, "errors": [], "createdResources": %s}}`, createdResources1))
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		Reply(200).
		BodyString(fmt.Sprintf(`{"id": "2", "status": {"completed": true, "failed": false, "errors": [], "createdResources": %s}}`, createdResources2))
	res, err := client.Batch("1", actions)
	assert.NoError(t, err)
	responses := res.Result.Get("responses").Array()
	assert.Equal(t, 2, len(responses))
	createdResources := res.Result.Get("status.createdResources").Array()
	assert.Equal(t, 110, len(createdResources))

	// Multi-batch Error (first ok, second error)
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		Reply(201).
		BodyString(`{"id": "1", "status": {"completed": true, "failed": false, "errors": []}}`)
	gock.New(client.BaseUrl).
		Post("/organizations/1/actionBatches").
		ReplyError(errors.New("fail"))
	actions = make([]ActionModel, 110)
	for i := range actions {
		actions[i] = NewAction("update", fmt.Sprintf("url%d", i), "{}")
	}
	res, err = client.Batch("1", actions)
	assert.Error(t, err)
	responses = res.Result.Get("responses").Array()
	assert.Equal(t, 2, len(responses))
	// First batch should be successful, second should be error object
	first := responses[0]
	second := responses[1]
	assert.False(t, first.Get("error").Exists(), "First batch should not be error")
	assert.True(t, second.Get("error").Exists(), "Second batch should be error")
	assert.Contains(t, second.Get("error").String(), "fail")
}
