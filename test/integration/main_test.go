// test/integration/main_test.go

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
	"web-service/internal/server"
)


func TestCommentAPI(t *testing.T) {

    t.Parallel()

    tests := []struct {
        name         string
        args         []string
        envVars      map[string]string
        setupFunc    func(t *testing.T)
        request      func(t *testing.T) (*http.Response, error)
        validateFunc func(t *testing.T, resp *http.Response)
    }{
        {
            name: "health check endpoint",
            args: []string{"server", "--port", "8081"},
            envVars: map[string]string{
				"JWT_SECRET":   "test-secret",
				"DATABASE_URL": "memory://test",
				"ENVIRONMENT":  "test",
			},
            request: func(t *testing.T) (*http.Response, error) {
                t.Log("Making health check request...")
                return http.Get("http://localhost:8081/healthz")
            },
            validateFunc: func(t *testing.T, resp *http.Response) {
                t.Logf("Validating health check response with status code: %d", resp.StatusCode)
                if resp.StatusCode != http.StatusOK {
                    t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
                }

                var response struct {
                    Status string `json:"status"`
                    Time   string `json:"time"`
                }

                if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
                    t.Fatal(err)
                }

                t.Logf("Received health check response: %+v", response)

                if response.Status != "ok" {
                    t.Errorf("expected status 'ok', got %q", response.Status)
                }
                if response.Time == "" {
                    t.Error("expected non-empty time")
                }
            },
        },
        {
            name: "create comment successfully",
            args: []string{"server", "--port", "8082"},
            envVars: map[string]string{
				"JWT_SECRET":   "test-secret",
				"DATABASE_URL": "memory://test",
				"ENVIRONMENT":  "test",
			},
            request: func(t *testing.T) (*http.Response, error) {
                t.Log("Making create comment request...")
                comment := struct {
                    Content string `json:"content"`
                    Author  string `json:"author"`
                }{
                    Content: "Test comment",
                    Author:  "Test author",
                }

                var buf bytes.Buffer
                if err := json.NewEncoder(&buf).Encode(comment); err != nil {
                    t.Fatal(err)
                }

                req, err := http.NewRequest(http.MethodPost, "http://localhost:8082/api/v1/comments", &buf)
                if err != nil {
                    t.Fatal(err)
                }

                // Login first to get a token
                loginReq := struct {
                    Username string `json:"username"`
                    Password string `json:"password"`
                }{
                    Username: "test",
                    Password: "test123",
                }

                var loginBuf bytes.Buffer
                if err := json.NewEncoder(&loginBuf).Encode(loginReq); err != nil {
                    t.Fatal(err)
                }

                loginResp, err := http.Post("http://localhost:8082/api/v1/login", "application/json", &loginBuf)
                if err != nil {
                    t.Fatal(err)
                }
                defer loginResp.Body.Close()

                var loginResult struct {
                    Token string `json:"token"`
                }
                if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
                    t.Fatal(err)
                }

                // Use the token for the comment creation request
                req.Header.Set("Authorization", "Bearer "+loginResult.Token)
                req.Header.Set("Content-Type", "application/json")

                return http.DefaultClient.Do(req)
            },
            validateFunc: func(t *testing.T, resp *http.Response) {
                t.Logf("Validating create comment response with status code: %d", resp.StatusCode)
                if resp.StatusCode != http.StatusCreated {
                    t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
                }

                var response struct {
                    ID        string    `json:"id"`
                    Content   string    `json:"content"`
                    Author    string    `json:"author"`
                    CreatedAt time.Time `json:"created_at"`
                    UserID    string    `json:"user_id,omitempty"`
                }

                if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
                    t.Fatal(err)
                }

                t.Logf("Received create comment response: %+v", response)

                if response.Content != "Test comment" {
                    t.Errorf("expected content %q, got %q", "Test comment", response.Content)
                }
                if response.Author != "Test author" {
                    t.Errorf("expected author %q, got %q", "Test author", response.Author)
                }
                if response.ID == "" {
                    t.Error("expected non-empty ID")
                }
                if response.CreatedAt.IsZero() {
                    t.Error("expected non-zero creation time")
                }
                if response.UserID == "" {
                    t.Error("expected non-empty user ID")
                }
            },
        },
        {
            name: "list comments",
            args: []string{"server", "--port", "8083"},
            envVars: map[string]string{
				"JWT_SECRET":   "test-secret",
				"DATABASE_URL": "memory://test",
				"ENVIRONMENT":  "test",
			},
            setupFunc: func(t *testing.T) {
                // Create a test comment first
                comment := struct {
                    Content string `json:"content"`
                    Author  string `json:"author"`
                }{
                    Content: "Setup comment",
                    Author:  "Setup author",
                }

                var buf bytes.Buffer
                if err := json.NewEncoder(&buf).Encode(comment); err != nil {
                    t.Fatal(err)
                }

                // Login to get token
                loginReq := struct {
                    Username string `json:"username"`
                    Password string `json:"password"`
                }{
                    Username: "test",
                    Password: "test123",
                }

                var loginBuf bytes.Buffer
                if err := json.NewEncoder(&loginBuf).Encode(loginReq); err != nil {
                    t.Fatal(err)
                }

                loginResp, err := http.Post("http://localhost:8083/api/v1/login", "application/json", &loginBuf)
                if err != nil {
                    t.Fatal(err)
                }
                defer loginResp.Body.Close()

                var loginResult struct {
                    Token string `json:"token"`
                }
                if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
                    t.Fatal(err)
                }

                req, err := http.NewRequest(http.MethodPost, "http://localhost:8083/api/v1/comments", &buf)
                if err != nil {
                    t.Fatal(err)
                }
                req.Header.Set("Authorization", "Bearer "+loginResult.Token)
                req.Header.Set("Content-Type", "application/json")

                resp, err := http.DefaultClient.Do(req)
                if err != nil {
                    t.Fatal(err)
                }
                defer resp.Body.Close()

                if resp.StatusCode != http.StatusCreated {
                    t.Fatalf("failed to create setup comment: status %d", resp.StatusCode)
                }
            },
            request: func(t *testing.T) (*http.Response, error) {
                t.Log("Making list comments request...")

                // Login to get token
                loginReq := struct {
                    Username string `json:"username"`
                    Password string `json:"password"`
                }{
                    Username: "test",
                    Password: "test123",
                }

                var loginBuf bytes.Buffer
                if err := json.NewEncoder(&loginBuf).Encode(loginReq); err != nil {
                    t.Fatal(err)
                }

                loginResp, err := http.Post("http://localhost:8083/api/v1/login", "application/json", &loginBuf)
                if err != nil {
                    t.Fatal(err)
                }
                defer loginResp.Body.Close()

                var loginResult struct {
                    Token string `json:"token"`
                }
                if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
                    t.Fatal(err)
                }

                req, err := http.NewRequest(http.MethodGet, "http://localhost:8083/api/v1/comments", nil)
                if err != nil {
                    t.Fatal(err)
                }
                req.Header.Set("Authorization", "Bearer "+loginResult.Token)

                return http.DefaultClient.Do(req)
            },
            validateFunc: func(t *testing.T, resp *http.Response) {
                t.Logf("Validating list comments response with status code: %d", resp.StatusCode)
                if resp.StatusCode != http.StatusOK {
                    t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
                }

                var response []struct {
                    ID        string    `json:"id"`
                    Content   string    `json:"content"`
                    Author    string    `json:"author"`
                    CreatedAt time.Time `json:"created_at"`
                    UserID    string    `json:"user_id,omitempty"`
                }

                if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
                    t.Fatal(err)
                }

                t.Logf("Received list comments response: %+v", response)

                if len(response) == 0 {
                    t.Error("expected non-empty comment list")
                }

                // Verify the setup comment is in the list
                found := false
                for _, comment := range response {
                    if comment.Content == "Setup comment" && comment.Author == "Setup author" {
                        found = true
                        break
                    }
                }
                if !found {
                    t.Error("setup comment not found in list")
                }
            },
        },
    }

    for _, tt := range tests {
        tt := tt // capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            t.Logf("Starting test: %s", tt.name)

            ctx, cancel := context.WithCancel(context.Background())
            t.Cleanup(func() {
                t.Log("Cleaning up test...")
                cancel()
            })

            var stdout, stderr bytes.Buffer

            getenv := func(key string) string {
                val := tt.envVars[key]
                t.Logf("Environment variable %s = %s", key, val)
                return val
            }

            // Start the server
            serverErrCh := make(chan error, 1)
            go func() {
                t.Log("Starting server...")
                if err := server.Run(ctx, &stdout, tt.args, getenv); err != nil {
                    select {
                    case <-ctx.Done():
                        // Expected error from shutdown
                        return
                    default:
                        serverErrCh <- err
                    }
                }
            }()

            // Wait for server to be ready
            endpoint := fmt.Sprintf("http://localhost:%s/healthz", tt.args[2])
            t.Logf("Waiting for server to be ready at %s", endpoint)
            if err := waitForReady(ctx, 5*time.Second, endpoint); err != nil {
                t.Logf("Server stdout:\n%s", stdout.String())
                t.Logf("Server stderr:\n%s", stderr.String())
                t.Fatalf("Server failed to become ready: %v", err)
            }

            // Check for any server errors
            select {
            case err := <-serverErrCh:
                if err != nil {
                    t.Fatalf("Server error: %v", err)
                }
            default:
                // Server still running, proceed with test
            }

            if tt.setupFunc != nil {
                t.Log("Running setup function...")
                tt.setupFunc(t)
            }

            t.Log("Making request...")
            resp, err := tt.request(t)
            if err != nil {
                t.Fatalf("Request failed: %v", err)
            }
            defer resp.Body.Close()

            t.Log("Validating response...")
            tt.validateFunc(t, resp)

            if t.Failed() {
                t.Logf("Server stdout:\n%s", stdout.String())
                t.Logf("Server stderr:\n%s", stderr.String())
            }
        })
    }
}

func waitForReady(ctx context.Context, timeout time.Duration, endpoint string) error {
    client := http.Client{
        Timeout: 1 * time.Second,
    }
    startTime := time.Now()

    for {
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
        if err != nil {
            return fmt.Errorf("failed to create request: %w", err)
        }

        resp, err := client.Do(req)
        if err != nil {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(250 * time.Millisecond):
                continue
            }
        }

        if resp != nil {
            io.Copy(io.Discard, resp.Body)
            resp.Body.Close()
            if resp.StatusCode == http.StatusOK {
                return nil
            }
        }

        if time.Since(startTime) >= timeout {
            return fmt.Errorf("timeout waiting for endpoint %s", endpoint)
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(250 * time.Millisecond):
        }
    }
}