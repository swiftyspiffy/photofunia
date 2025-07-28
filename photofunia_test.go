package photofunia

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

type MockLogger struct {
	DebugMessages []string
	InfoMessages  []string
}

func (l *MockLogger) Debug(msg string, fields ...Field) {
	l.DebugMessages = append(l.DebugMessages, msg)
}

func (l *MockLogger) Info(msg string, fields ...Field) {
	l.InfoMessages = append(l.InfoMessages, msg)
}

type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestNewPhotoFuniaClient(t *testing.T) {
	client := NewPhotoFuniaClient()

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	_, ok := client.logger.(NoopLogger)
	if !ok {
		t.Errorf("Expected NoopLogger, got %T", client.logger)
	}

	if client.client == nil {
		t.Error("Expected non-nil http.Client")
	}
}

func TestNewPhotoFuniaClientWithLogger(t *testing.T) {
	logger := &MockLogger{}
	client := NewPhotoFuniaClientWithLogger(logger)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.logger != logger {
		t.Errorf("Expected logger to be our MockLogger")
	}

	if client.client == nil {
		t.Error("Expected non-nil http.Client")
	}
}

func TestExtractImageURL(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		want        string
		wantErr     bool
	}{
		{
			name: "Valid HTML with image",
			htmlContent: `<html><body><img id="result-image" src="https://example.com/image.jpg" alt="Result"></body></html>`,
			want: "https://example.com/image.jpg",
			wantErr: false,
		},
		{
			name: "HTML without image tag",
			htmlContent: `<html><body><div>No image here</div></body></html>`,
			want: "",
			wantErr: true,
		},
		{
			name: "Image tag without src attribute",
			htmlContent: `<html><body><img id="result-image" alt="Result"></body></html>`,
			want: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractImageURL([]byte(tt.htmlContent))

			if (err != nil) != tt.wantErr {
				t.Errorf("extractImageURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("extractImageURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFatify(t *testing.T) {
	mockTransport := &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "cookie-warning") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Set-Cookie": []string{"PHPSESSID=test-session-id"}},
				}, nil
			}

			if strings.Contains(req.URL.String(), "/images") {
				jsonResponse := `{"response":{"key":"test-image-key","server":1,"existed":false,"expiry":0,"created":0,"lifetime":0,"image":{"highres":{"url":"","width":0,"height":0},"preview":{"url":"","width":0,"height":0},"thumb":{"url":"","width":0,"height":0}},"sid":"test-sid"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(jsonResponse)),
				}, nil
			}

			if strings.Contains(req.URL.String(), "faces/fat_maker") && req.Method == "POST" {
				resultURL, _ := url.Parse("https://photofunia.com/results/result123")
				return &http.Response{
					StatusCode: http.StatusOK,
					Request:    &http.Request{URL: resultURL},
				}, nil
			}

			if req.Method == "GET" && strings.Contains(req.URL.String(), "/results/") {
				htmlContent := `<html><body><img id="result-image" src="https://example.com/result.jpg" alt="Result"></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(htmlContent)),
				}, nil
			}

			if strings.Contains(req.URL.String(), "example.com/result.jpg") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("fake-image-data"))),
				}, nil
			}

			return nil, errors.New("unexpected request")
		},
	}

	client := &PhotoFuniaClient{
		logger: &MockLogger{},
		client: &http.Client{
			Transport: mockTransport,
		},
	}

	imageData := []byte("fake-image-data")
	imageReader := io.NopCloser(bytes.NewReader(imageData))

	result, err := client.Fatify(imageReader)

	if err != nil {
		t.Fatalf("Fatify() error = %v", err)
	}

	if string(result) != "fake-image-data" {
		t.Errorf("Fatify() = %v, want %v", string(result), "fake-image-data")
	}
}

func TestClownify(t *testing.T) {
	tests := []struct {
		name       string
		includeHat bool
	}{
		{
			name:       "Clownify without hat",
			includeHat: false,
		},
		{
			name:       "Clownify with hat",
			includeHat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := &MockTransport{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.String(), "cookie-warning") {
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Set-Cookie": []string{"PHPSESSID=test-session-id"}},
						}, nil
					}

					if strings.Contains(req.URL.String(), "/images") {
						jsonResponse := `{"response":{"key":"test-image-key","server":1,"existed":false,"expiry":0,"created":0,"lifetime":0,"image":{"highres":{"url":"","width":0,"height":0},"preview":{"url":"","width":0,"height":0},"thumb":{"url":"","width":0,"height":0}},"sid":"test-sid"}}`
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(jsonResponse)),
						}, nil
					}

					if strings.Contains(req.URL.String(), "all_effects/clown") && req.Method == "POST" {
						body, _ := io.ReadAll(req.Body)
						bodyStr := string(body)

						req.Body = io.NopCloser(bytes.NewReader(body))

						hatNamePattern := `name="hat"`
						if !strings.Contains(bodyStr, hatNamePattern) {
							t.Errorf("Expected hat parameter in request body")
						} else {
							hatValuePattern := ""
							if tt.includeHat {
								hatValuePattern = `name="hat"[\s\S]*?on`
							} else {
								hatValuePattern = `name="hat"[\s\S]*?off`
							}

							matched, _ := regexp.MatchString(hatValuePattern, bodyStr)
							if !matched {
								if tt.includeHat {
									t.Errorf("Expected hat=on in request body")
								} else {
									t.Errorf("Expected hat=off in request body")
								}
							}
						}

						resultURL, _ := url.Parse("https://photofunia.com/results/result123")
						return &http.Response{
							StatusCode: http.StatusOK,
							Request:    &http.Request{URL: resultURL},
						}, nil
					}

					if req.Method == "GET" && strings.Contains(req.URL.String(), "/results/") {
						htmlContent := `<html><body><img id="result-image" src="https://example.com/result.jpg" alt="Result"></body></html>`
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(htmlContent)),
						}, nil
					}

					if strings.Contains(req.URL.String(), "example.com/result.jpg") {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewReader([]byte("fake-image-data"))),
						}, nil
					}

					return nil, errors.New("unexpected request")
				},
			}

			client := &PhotoFuniaClient{
				logger: &MockLogger{},
				client: &http.Client{
					Transport: mockTransport,
				},
			}

			imageData := []byte("fake-image-data")
			imageReader := io.NopCloser(bytes.NewReader(imageData))

			result, err := client.Clownify(imageReader, tt.includeHat)

			if err != nil {
				t.Fatalf("Clownify() error = %v", err)
			}

			if string(result) != "fake-image-data" {
				t.Errorf("Clownify() = %v, want %v", string(result), "fake-image-data")
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		mockResponses func(req *http.Request) (*http.Response, error)
		wantErr       bool
		errorContains string
	}{
		{
			name: "Session ID generation fails",
			mockResponses: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.String(), "cookie-warning") {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Status:     "500 Internal Server Error",
					}, nil
				}
				return nil, errors.New("unexpected request")
			},
			wantErr:       true,
			errorContains: "server returned non-OK status",
		},
		{
			name: "Image upload fails",
			mockResponses: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.String(), "cookie-warning") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Set-Cookie": []string{"PHPSESSID=test-session-id"}},
					}, nil
				}
				if strings.Contains(req.URL.String(), "/images") {
					return nil, errors.New("network error")
				}
				return nil, errors.New("unexpected request")
			},
			wantErr:       true,
			errorContains: "network error",
		},
		{
			name: "Invalid JSON response",
			mockResponses: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.String(), "cookie-warning") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Set-Cookie": []string{"PHPSESSID=test-session-id"}},
					}, nil
				}
				if strings.Contains(req.URL.String(), "/images") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("invalid json")),
					}, nil
				}
				return nil, errors.New("unexpected request")
			},
			wantErr:       true,
			errorContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := &MockTransport{
				RoundTripFunc: tt.mockResponses,
			}

			client := &PhotoFuniaClient{
				logger: &MockLogger{},
				client: &http.Client{
					Transport: mockTransport,
				},
			}

			imageData := []byte("fake-image-data")
			imageReader := io.NopCloser(bytes.NewReader(imageData))

			_, err := client.Fatify(imageReader)

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err != nil)
			}

			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Error message does not contain expected text. Got: %v, want to contain: %v", err.Error(), tt.errorContains)
			}
		})
	}
}