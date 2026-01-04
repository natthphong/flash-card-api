// Package adapter/home_proxy.go
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
)

type Adapter struct {
	BaseURL string
	Client  *http.Client
}

type Config struct {
	BaseURL string
	Timeout time.Duration
}

func NewAdapter(cfg config.AdapterConfig) (*Adapter, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("BASEURL is required")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid BASEURL: %w", err)
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("BASEURL must include scheme (http/https)")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &Adapter{
		BaseURL: strings.TrimRight(cfg.BaseURL, "/"),
		Client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// ---------- Options / Payloads ----------

type RequestOptions struct {
	Headers map[string]string
	Query   map[string]string

	// One of these payloads can be set:
	JSON any
	Form *FormData

	// If you want raw body yourself:
	Body        io.Reader
	ContentType string
}

type FormData struct {
	Fields map[string]string
	Files  []FormFile
}

type FormFile struct {
	FieldName   string
	FileName    string
	ContentType string
	Reader      io.Reader
}

// ---------- Public Methods (GET/POST/PUT/DELETE) ----------

func (a *Adapter) Get(ctx context.Context, p string, opt *RequestOptions) (*http.Response, []byte, error) {
	return a.do(ctx, http.MethodGet, p, opt)
}

func (a *Adapter) Post(ctx context.Context, p string, opt *RequestOptions) (*http.Response, []byte, error) {
	return a.do(ctx, http.MethodPost, p, opt)
}

func (a *Adapter) Put(ctx context.Context, p string, opt *RequestOptions) (*http.Response, []byte, error) {
	return a.do(ctx, http.MethodPut, p, opt)
}

func (a *Adapter) Delete(ctx context.Context, p string, opt *RequestOptions) (*http.Response, []byte, error) {
	return a.do(ctx, http.MethodDelete, p, opt)
}

// ---------- Core ----------

func (a *Adapter) do(ctx context.Context, method, p string, opt *RequestOptions) (*http.Response, []byte, error) {
	if opt == nil {
		opt = &RequestOptions{}
	}

	fullURL, err := a.buildURL(p, opt.Query)
	if err != nil {
		return nil, nil, err
	}

	body, contentType, err := a.buildBody(opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, nil, err
	}

	// headers
	for k, v := range opt.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	// read body (caller can still use resp if needed)
	defer resp.Body.Close()
	b, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp, nil, readErr
	}

	// treat non-2xx as error (customize if you want)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, b, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return resp, b, nil
}

func (a *Adapter) buildURL(pth string, query map[string]string) (string, error) {
	base, err := url.Parse(a.BaseURL)
	if err != nil {
		return "", err
	}

	// join path safely
	joined := path.Join(base.Path, strings.TrimPrefix(pth, "/"))
	base.Path = joined

	q := base.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	base.RawQuery = q.Encode()

	return base.String(), nil
}

func (a *Adapter) buildBody(opt *RequestOptions) (io.Reader, string, error) {
	// If caller supplies raw body, use it
	if opt.Body != nil {
		return opt.Body, opt.ContentType, nil
	}

	// JSON
	if opt.JSON != nil {
		b, err := json.Marshal(opt.JSON)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), "application/json", nil
	}

	// multipart form-data
	if opt.Form != nil {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)

		// fields
		for k, v := range opt.Form.Fields {
			_ = w.WriteField(k, v)
		}

		// files
		for _, f := range opt.Form.Files {
			if f.FieldName == "" || f.FileName == "" || f.Reader == nil {
				continue
			}

			var fw io.Writer
			if f.ContentType != "" {
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(f.FieldName), escapeQuotes(f.FileName)))
				h.Set("Content-Type", f.ContentType)
				var err error
				fw, err = w.CreatePart(h)
				if err != nil {
					_ = w.Close()
					return nil, "", err
				}
			} else {
				var err error
				fw, err = w.CreateFormFile(f.FieldName, f.FileName)
				if err != nil {
					_ = w.Close()
					return nil, "", err
				}
			}

			if _, err := io.Copy(fw, f.Reader); err != nil {
				_ = w.Close()
				return nil, "", err
			}
		}

		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return bytes.NewReader(buf.Bytes()), w.FormDataContentType(), nil
	}

	// no body
	return nil, "", nil
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
