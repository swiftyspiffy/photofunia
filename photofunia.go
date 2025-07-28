// Package photofunia provides a client for the PhotoFunia service,
// allowing programmatic application of various image effects to photos.
//
// This package handles all the necessary HTTP requests, session management,
// and image processing to interact with the PhotoFunia web service.
package photofunia

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const (
	baseURL         = "https://photofunia.com"
	defaultBoundary = "----WebKitFormBoundaryL3VFyS6LkNI3s7UM"
	userAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
)

// PhotoFuniaClient is a client for the PhotoFunia service.
// It handles session management and HTTP requests to apply various effects to images.
type PhotoFuniaClient struct {
	PHPSESSID string
	logger    Logger
	client    *http.Client
	timeout   time.Duration
}

// DefaultTimeout is the default timeout for HTTP requests.
const DefaultTimeout = 30 * time.Second

// NewPhotoFuniaClient creates a new PhotoFuniaClient with a no-op logger.
// This is a convenience function for users who don't need logging.
func NewPhotoFuniaClient() *PhotoFuniaClient {
	return NewPhotoFuniaClientWithLogger(NoopLogger{})
}

// NewPhotoFuniaClientWithLogger creates a new PhotoFuniaClient with the provided logger.
// This allows users to integrate the client with their own logging system.
func NewPhotoFuniaClientWithLogger(logger Logger) *PhotoFuniaClient {
	return &PhotoFuniaClient{
		logger:  logger,
		client:  &http.Client{Timeout: DefaultTimeout},
		timeout: DefaultTimeout,
	}
}

// WithTimeout sets a custom timeout for HTTP requests.
// Returns a new client with the specified timeout.
func (c *PhotoFuniaClient) WithTimeout(timeout time.Duration) *PhotoFuniaClient {
	newClient := *c
	newClient.timeout = timeout
	newClient.client = &http.Client{Timeout: timeout}
	return &newClient
}

// FatifyWithContext applies the "fat maker" effect to the provided image with context support.
// It returns the processed image data as a byte slice.
//
// The ctx parameter allows for cancellation and timeout control.
// The img parameter should be an io.ReadCloser containing the image data.
// The function will close the reader when done.
func (c *PhotoFuniaClient) FatifyWithContext(ctx context.Context, img io.ReadCloser) ([]byte, error) {
	params := map[string]string{
		"current-category": "faces",
		"image:crop":       "0.0.961.1093",
		"size":             "XXXXXL",
	}

	return c.applyEffectWithContext(ctx, img, "faces/fat_maker", params, "fatify")
}

// ClownifyWithContext applies the clown effect to the provided image with context support.
// It returns the processed image data as a byte slice.
//
// The ctx parameter allows for cancellation and timeout control.
// The img parameter should be an io.ReadCloser containing the image data.
// The function will close the reader when done.
//
// The includeHat parameter determines whether a clown hat is added to the image.
func (c *PhotoFuniaClient) ClownifyWithContext(ctx context.Context, img io.ReadCloser, includeHat bool) ([]byte, error) {
	params := map[string]string{
		"current-category": "all_effects",
		"image:crop":       "0.0.961.1093",
	}

	if includeHat {
		params["hat"] = "on"
	} else {
		params["hat"] = "off"
	}

	return c.applyEffectWithContext(ctx, img, "all_effects/clown", params, "clownify")
}

func (c *PhotoFuniaClient) applyEffect(img io.ReadCloser, effectPath string, params map[string]string, effectName string) ([]byte, error) {
	return c.applyEffectWithContext(context.Background(), img, effectPath, params, effectName)
}

func (c *PhotoFuniaClient) applyEffectWithContext(ctx context.Context, img io.ReadCloser, effectPath string, params map[string]string, effectName string) ([]byte, error) {
	response, err := c.uploadImageWithContext(ctx, img)
	if err != nil {
		return nil, err
	}

	imageKey := response.Response.Key
	if imageKey == "" {
		return nil, errors.New("image key is empty in the response")
	}

	c.logger.Info("got image key", Field{"key", imageKey})

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	writer.SetBoundary(defaultBoundary)

	params["image"] = imageKey

	for key, value := range params {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/categories/%s?server=1", baseURL, effectPath)
	req, err := c.createRequestWithContext(ctx, "POST", url, &requestBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "multipart/form-data; boundary="+defaultBoundary)

	req.Header.Set("Referer", fmt.Sprintf("%s/categories/%s", baseURL, effectPath))

	c.logger.Info(fmt.Sprintf("sending request to PhotoFunia %s effect", effectName))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to PhotoFunia %s effect: %w", effectName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status: %s", resp.Status)
	}

	c.logger.Info(fmt.Sprintf("successfully received response from PhotoFunia %s effect", effectName),
		Field{"contentType", resp.Header.Get("Content-Type")},
		Field{"contentLength", resp.ContentLength},
		Field{"resultURL", resp.Request.URL.String()})

	return c.getResultImageWithContext(ctx, resp.Request.URL.String())
}

func (c *PhotoFuniaClient) generateSessIDWithContext(ctx context.Context) error {
	c.logger.Info("generating new PHPSESSID")

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/cookie-warning", nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for PHPSESSID: %w", err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", "accept_cookie=true")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request to PhotoFunia for PHPSESSID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK status for PHPSESSID request: %s", resp.Status)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "PHPSESSID" {
			c.PHPSESSID = cookie.Value
			return nil
		}
	}

	return errors.New("PHPSESSID cookie not found in response")
}

func (c *PhotoFuniaClient) getResultImageWithContext(ctx context.Context, resultURL string) ([]byte, error) {
	req, err := c.createRequestWithContext(ctx, "GET", resultURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for result page: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get result page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status for result page: %s", resp.Status)
	}

	htmlContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML content: %w", err)
	}

	imageURL, err := extractImageURL(htmlContent)
	if err != nil {
		return nil, err
	}

	c.logger.Info("found image URL", Field{"url", imageURL})

	imgReq, err := c.createRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for image: %w", err)
	}

	imgReq.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	imgReq.Header.Set("Referer", resultURL)

	imgResp, err := c.client.Do(imgReq)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status for image: %s", imgResp.Status)
	}

	imageData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	c.logger.Info("successfully downloaded image", Field{"url", imageURL}, Field{"size", len(imageData)})
	return imageData, nil
}

func extractImageURL(htmlContent []byte) (string, error) {
	imgTagStart := `<img id="result-image"`
	srcAttrStart := `src="`
	srcAttrEnd := `"`

	imgTagIndex := bytes.Index(htmlContent, []byte(imgTagStart))
	if imgTagIndex == -1 {
		return "", errors.New("could not find result image in HTML")
	}

	srcStartIndex := bytes.Index(htmlContent[imgTagIndex:], []byte(srcAttrStart))
	if srcStartIndex == -1 {
		return "", errors.New("could not find src attribute in image tag")
	}
	srcStartIndex += imgTagIndex + len(srcAttrStart)

	srcEndIndex := bytes.Index(htmlContent[srcStartIndex:], []byte(srcAttrEnd))
	if srcEndIndex == -1 {
		return "", errors.New("could not find end of src attribute")
	}
	srcEndIndex += srcStartIndex

	return string(htmlContent[srcStartIndex:srcEndIndex]), nil
}

func (c *PhotoFuniaClient) uploadImageWithContext(ctx context.Context, imageReader io.ReadCloser) (*photoFuniaResponse, error) {
	defer imageReader.Close()

	imageData, err := io.ReadAll(imageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	c.logger.Info("read image data", Field{"size", len(imageData)})

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	uploadBoundary := "----WebKitFormBoundaryx4CBHpJEw9pPEXE4"
	writer.SetBoundary(uploadBoundary)

	part, err := writer.CreateFormFile("image", "image.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = part.Write(imageData); err != nil {
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}

	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := c.createRequestWithContext(ctx, "POST", baseURL+"/images?server=1", &requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+uploadBoundary)
	req.Header.Set("Referer", baseURL+"/categories/all_effects/clown")

	c.logger.Info("sending request to PhotoFunia")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to PhotoFunia: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status: %s", resp.Status)
	}

	c.logger.Info("successfully received response from PhotoFunia",
		Field{"contentType", resp.Header.Get("Content-Type")},
		Field{"contentLength", resp.ContentLength})

	var photoFuniaResp photoFuniaResponse
	if err = json.NewDecoder(resp.Body).Decode(&photoFuniaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &photoFuniaResp, nil
}

func (c *PhotoFuniaClient) createRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("User-Agent", userAgent)

	if c.PHPSESSID == "" {
		if err := c.generateSessIDWithContext(ctx); err != nil {
			return nil, err
		}
	}

	req.Header.Set("Cookie", fmt.Sprintf("accept_cookie=true; PHPSESSID=%s", c.PHPSESSID))

	return req, nil
}
