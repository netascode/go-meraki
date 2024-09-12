// Package meraki is a Cisco Meraki REST client library for Go.
package meraki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/juju/ratelimit"
)

const DefaultMaxRetries int = 3
const DefaultBackoffMinDelay int = 2
const DefaultBackoffMaxDelay int = 60
const DefaultBackoffDelayFactor float64 = 3

// Client is an HTTP Meraki client.
// Use meraki.NewClient to initiate a client.
// This will ensure proper cookie handling and processing of modifiers.
//
// Requests are protected from concurrent writing (concurrent DELETE/POST/PUT),
// across all API paths. Any GET requests, or requests from different clients
// are not protected against concurrent writing.
type Client struct {
	// HttpClient is the *http.Client used for API requests
	HttpClient *http.Client
	// BaseUrl is the Meraki Dashboard Base API Url, default is https://api.meraki.com/api/v1
	BaseUrl string
	// ApiToken is the current API token
	ApiToken string
	// UserAgent is the HTTP User-Agent string
	UserAgent string
	// Maximum number of requests per second
	RequestPerSecond int
	// Maximum number of retries
	MaxRetries int
	// Minimum delay between two retries
	BackoffMinDelay int
	// Maximum delay between two retries
	BackoffMaxDelay int
	// Backoff delay factor
	BackoffDelayFactor float64

	RateLimiterBucket *ratelimit.Bucket
}

// NewClient creates a new Meraki HTTP client.
// Pass modifiers in to modify the behavior of the client, e.g.
//
//	client, _ := NewClient("abc123", RequestTimeout(120))
func NewClient(token string, mods ...func(*Client)) (Client, error) {
	cookieJar, _ := cookiejar.New(nil)
	httpClient := http.Client{
		Timeout: 60 * time.Second,
		Jar:     cookieJar,
	}

	client := Client{
		HttpClient:         &httpClient,
		BaseUrl:            "https://api.meraki.com/api/v1",
		ApiToken:           token,
		UserAgent:          "go-meraki netascode",
		MaxRetries:         DefaultMaxRetries,
		BackoffMinDelay:    DefaultBackoffMinDelay,
		BackoffMaxDelay:    DefaultBackoffMaxDelay,
		BackoffDelayFactor: DefaultBackoffDelayFactor,
		RateLimiterBucket:  ratelimit.NewBucketWithQuantum(time.Second, int64(10), int64(10)),
	}

	for _, mod := range mods {
		mod(&client)
	}
	return client, nil
}

// BaseUrl modifies the API base URL. Default value is 'https://api.meraki.com/api/v1'.
func BaseUrl(x string) func(*Client) {
	return func(client *Client) {
		client.BaseUrl = x
	}
}

// UserAgent modifies the HTTP user agent string. Default value is 'go-meraki netascode'.
func UserAgent(x string) func(*Client) {
	return func(client *Client) {
		client.UserAgent = x
	}
}

// RequestPerSecond modifies the maximum number of requests per second. Default value is 10.
func RequestPerSecond(x int) func(*Client) {
	return func(client *Client) {
		client.RateLimiterBucket = ratelimit.NewBucketWithQuantum(time.Second, int64(x), int64(x))
	}
}

// RequestTimeout modifies the HTTP request timeout from the default of 60 seconds.
func RequestTimeout(x time.Duration) func(*Client) {
	return func(client *Client) {
		client.HttpClient.Timeout = x * time.Second
	}
}

// MaxRetries modifies the maximum number of retries from the default of 3.
func MaxRetries(x int) func(*Client) {
	return func(client *Client) {
		client.MaxRetries = x
	}
}

// BackoffMinDelay modifies the minimum delay between two retries from the default of 2.
func BackoffMinDelay(x int) func(*Client) {
	return func(client *Client) {
		client.BackoffMinDelay = x
	}
}

// BackoffMaxDelay modifies the maximum delay between two retries from the default of 60.
func BackoffMaxDelay(x int) func(*Client) {
	return func(client *Client) {
		client.BackoffMaxDelay = x
	}
}

// BackoffDelayFactor modifies the backoff delay factor from the default of 3.
func BackoffDelayFactor(x float64) func(*Client) {
	return func(client *Client) {
		client.BackoffDelayFactor = x
	}
}

// NewReq creates a new Req request for this client.
func (client Client) NewReq(method, uri string, body io.Reader, mods ...func(*Req)) Req {
	httpReq, _ := http.NewRequest(method, client.BaseUrl+uri, body)
	req := Req{
		HttpReq:    httpReq,
		LogPayload: true,
	}
	for _, mod := range mods {
		mod(&req)
	}
	return req
}

func logJson(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var err error
	var pretty []byte
	if body[0] == '{' {
		m := make(map[string]interface{})
		err = json.Unmarshal(body, &m)
		if err != nil {
			return err
		}
		pretty, err = json.MarshalIndent(m, "", "  ")
		if err != nil {
			return err
		}
	}
	if body[0] == '[' {
		a := make([]interface{}, 0)
		err = json.Unmarshal(body, &a)
		if err != nil {
			return err
		}
		pretty, err = json.MarshalIndent(a, "", "  ")
		if err != nil {
			return err
		}
	}
	for _, l := range strings.Split(string(pretty), "\n") {
		log.Println(l)
	}
	return nil
}

// Do makes a request.
// Requests for Do are built ouside of the client, e.g.
//
//	req := client.NewReq("GET", "/organizations", nil)
//	res, _ := client.Do(req)
func (client *Client) Do(req Req) (Res, error) {
	// add token
	req.HttpReq.Header.Add("Authorization", "Bearer "+client.ApiToken)
	req.HttpReq.Header.Add("User-Agent", client.UserAgent)
	req.HttpReq.Header.Add("Content-Type", "application/json")
	req.HttpReq.Header.Add("Accept", "application/json")
	// retain the request body across multiple attempts
	var body []byte
	if req.HttpReq.Body != nil {
		body, _ = io.ReadAll(req.HttpReq.Body)
	}

	var res Res

	for attempts := 0; ; attempts++ {
		client.RateLimiterBucket.Wait(1) // Block until rate limit token available

		req.HttpReq.Body = io.NopCloser(bytes.NewBuffer(body))
		if req.LogPayload {
			log.Println("REQUEST --------------------------")
			log.Printf("%s %s\n", req.HttpReq.Method, req.HttpReq.URL)
			for k, v := range req.HttpReq.Header {
				if k != "Authorization" {
					log.Printf("%s: %s\n", k, v)
				} else {
					log.Println("Authorization: ****")
				}
			}
			log.Println("--------------------------")

			err := logJson(body)
			if err != nil {
				log.Printf("failed to log json request: %s\n", err.Error())
			}

		} else {
			log.Printf("[DEBUG] HTTP Request: %s, %s", req.HttpReq.Method, req.HttpReq.URL)
		}

		httpRes, err := client.HttpClient.Do(req.HttpReq)
		if err != nil {
			if ok := client.Backoff(attempts); !ok {
				log.Printf("[ERROR] HTTP Connection error occured: %+v", err)
				log.Printf("[DEBUG] Exit from Do method")
				return Res{}, err
			} else {
				log.Printf("[ERROR] HTTP Connection failed: %s, retries: %v", err, attempts)
				continue
			}
		}

		defer httpRes.Body.Close()
		bodyBytes, err := io.ReadAll(httpRes.Body)
		if err != nil {
			if ok := client.Backoff(attempts); !ok {
				log.Printf("[ERROR] Cannot decode response body: %+v", err)
				log.Printf("[DEBUG] Exit from Do method")
				return Res{}, err
			} else {
				log.Printf("[ERROR] Cannot decode response body: %s, retries: %v", err, attempts)
				continue
			}
		}
		res = Res(gjson.ParseBytes(bodyBytes))
		if req.LogPayload {
			log.Printf("RESPONSE %d --------------------------\n", httpRes.StatusCode)
			err := logJson([]byte(res.Raw))
			log.Println("--------------------------")
			if err != nil {
				log.Printf("failed to log json response: %s\n", err.Error())
			}
		}

		if httpRes.StatusCode >= 200 && httpRes.StatusCode <= 299 {
			log.Printf("[DEBUG] Exit from Do method")
			break
		} else {
			if ok := client.Backoff(attempts); !ok {
				log.Printf("[ERROR] HTTP Request failed: StatusCode %v", httpRes.StatusCode)
				log.Printf("[DEBUG] Exit from Do method")
				return res, fmt.Errorf("HTTP Request failed: StatusCode %v", httpRes.StatusCode)
			} else if httpRes.StatusCode == 429 {
				retryAfter := httpRes.Header.Get("Retry-After")
				retryAfterDuration := time.Duration(0)
				if retryAfter == "0" {
					retryAfterDuration = time.Second
				} else if retryAfter != "" {
					retryAfterDuration, _ = time.ParseDuration(retryAfter + "s")
				} else {
					retryAfterDuration = 15 * time.Second
				}
				log.Printf("[WARNING] HTTP Request rate limited, waiting %v seconds, Retries: %v", retryAfterDuration.Seconds(), attempts)
				time.Sleep(retryAfterDuration)
				continue
			} else if httpRes.StatusCode >= 500 && httpRes.StatusCode <= 599 {
				log.Printf("[ERROR] HTTP Request failed: StatusCode %v, Retries: %v", httpRes.StatusCode, attempts)
				continue
			} else {
				log.Printf("[ERROR] HTTP Request failed: StatusCode %v", httpRes.StatusCode)
				log.Printf("[DEBUG] Exit from Do method")
				return res, fmt.Errorf("HTTP Request failed: StatusCode %v", httpRes.StatusCode)
			}
		}
	}

	// Return JSON error message if present
	if res.Get("errors").Exists() && len(res.Get("errors").Array()) > 0 {
		log.Printf("[ERROR] JSON error: %s", res.Get("errors").String())
		return res, fmt.Errorf("JSON error: %s", res.Get("errors").String())
	}
	return res, nil
}

// Get makes a GET request and returns a GJSON result.
// Results will be the raw data structure as returned by Meraki API
func (client *Client) Get(path string, mods ...func(*Req)) (Res, error) {
	req := client.NewReq("GET", path, nil, mods...)
	return client.Do(req)
}

// Delete makes a DELETE request.
func (client *Client) Delete(path string, mods ...func(*Req)) (Res, error) {
	req := client.NewReq("DELETE", path, nil, mods...)
	return client.Do(req)
}

// Post makes a POST request and returns a GJSON result.
// Hint: Use the Body struct to easily create POST body data.
func (client *Client) Post(path, data string, mods ...func(*Req)) (Res, error) {
	req := client.NewReq("POST", path, strings.NewReader(data), mods...)
	return client.Do(req)
}

// Put makes a PUT request and returns a GJSON result.
// Hint: Use the Body struct to easily create PUT body data.
func (client *Client) Put(path, data string, mods ...func(*Req)) (Res, error) {
	req := client.NewReq("PUT", path, strings.NewReader(data), mods...)
	return client.Do(req)
}

// Backoff waits following an exponential backoff algorithm
func (client *Client) Backoff(attempts int) bool {
	log.Printf("[DEBUG] Beginning backoff method: attempt %v of %v", attempts, client.MaxRetries)
	if attempts >= client.MaxRetries {
		log.Printf("[DEBUG] Exit from backoff method with return value false")
		return false
	}

	minDelay := time.Duration(client.BackoffMinDelay) * time.Second
	maxDelay := time.Duration(client.BackoffMaxDelay) * time.Second

	min := float64(minDelay)
	backoff := min * math.Pow(client.BackoffDelayFactor, float64(attempts))
	if backoff > float64(maxDelay) {
		backoff = float64(maxDelay)
	}
	backoff = (rand.Float64()/2+0.5)*(backoff-min) + min
	backoffDuration := time.Duration(backoff)
	log.Printf("[TRACE] Starting sleeping for %v", backoffDuration.Round(time.Second))
	time.Sleep(backoffDuration)
	log.Printf("[DEBUG] Exit from backoff method with return value true")
	return true
}
