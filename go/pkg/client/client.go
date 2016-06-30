package client

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	// FormatCSV returns the document in Comma Separated Values format
	FormatCSV = "csv"
	// FormatXML returns the document in an XML format using HTML tables.
	FormatXML = "xml"
	// FormatXLSX is an alias for FormatXLSXSinglePage
	FormatXLSX = FormatXLSXSinglePage
	// FormatXLSXSinglePage returns the document in a single-sheet
	// Excel file.
	FormatXLSXSinglePage = "xlsx-single"
	// FormatXLSXMultiplePages returns the document in a multi-sheet Excel
	// file, one sheet per page.
	FormatXLSXMultiplePages = "xlsx-multiple"
)

// Client represents an (endPoint, apiKey) and provides configuration of the
// http.Client.
type Client struct {
	EndPoint   string
	APIKey     string
	HTTPClient *http.Client
}

// DefaultClient provides a client with a usable default configuration.
var DefaultClient = &Client{
	EndPoint:   os.Getenv("PDFTABLES_ENDPOINT"),
	APIKey:     os.Getenv("PDFTABLES_API_KEY"),
	HTTPClient: http.DefaultClient,
}

type namedReader interface {
	Name() string
	io.Reader
}

func (c *Client) url(format string) (string, error) {
	endPoint := "https://pdftables.com/api"
	if c.EndPoint != "" {
		endPoint = c.EndPoint
	}

	u, err := url.Parse(endPoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Add("key", c.APIKey)
	q.Add("format", format)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ErrHTTP represents a minimal non-200 HTTP response.
type ErrHTTP struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *ErrHTTP) Error() string {
	return fmt.Sprintf("Non-200 response: %d %s: %q", e.StatusCode, e.Status, e.Body)
}

// Do makes a request to the PDFTables.com API using the DefaultClient.
func Do(in namedReader, format string) (io.ReadCloser, error) {
	return DefaultClient.Do(in, format)
}

// Do uploads a PDF and returns the result converted into the desired format.
// Note: *os.File satisfies namedReader.
func (c *Client) Do(in namedReader, format string) (io.ReadCloser, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	u, err := c.url(format)
	if err != nil {
		return nil, err
	}

	r, err := NewPOSTMultipartBodyRequest(u, in)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var buf [1024]byte
		n, _ := resp.Body.Read(buf[:])
		msg := string(buf[:n])
		resp.Body.Close()
		return nil, &ErrHTTP{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       msg,
		}
	}
	return resp.Body, nil
}

// NewPOSTMultipartBodyRequest returns a new HTTP request which POSTS `in` to `url`.
func NewPOSTMultipartBodyRequest(url string, in namedReader) (*http.Request, error) {
	contentType, multipartBody := newMultipartBody(in)
	r, err := http.NewRequest("POST", url, multipartBody)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", contentType)
	return r, nil
}

func newMultipartBody(in namedReader) (string, io.ReadCloser) {
	pr, pw := io.Pipe()
	mpw := multipart.NewWriter(pw)

	go func() {
		var err error
		defer pw.CloseWithError(err)

		defer mpw.Close()

		var out io.Writer
		out, err = mpw.CreateFormFile("file", filepath.Base(in.Name()))
		if err == nil {
			// err is passed out via defer CloseWithError
			_, err = io.Copy(out, in)
		}
	}()
	return mpw.FormDataContentType(), pr
}
