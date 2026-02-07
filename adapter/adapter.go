// Package adapter/home_proxy.go
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"path/filepath"
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
	if opt == nil {
		return nil, "", nil
	}

	// Raw body
	if opt.Body != nil {
		return opt.Body, strings.TrimSpace(opt.ContentType), nil
	}

	if opt.Form != nil {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)

		// fields
		for k, v := range opt.Form.Fields {
			if strings.TrimSpace(k) == "" {
				continue
			}
			if err := w.WriteField(k, v); err != nil {
				_ = w.Close()
				return nil, "", fmt.Errorf("write field %q: %w", k, err)
			}
		}

		// files
		for i, f := range opt.Form.Files {
			if strings.TrimSpace(f.FieldName) == "" {
				_ = w.Close()
				return nil, "", fmt.Errorf("form file[%d]: FieldName is required", i)
			}
			if strings.TrimSpace(f.FileName) == "" {
				_ = w.Close()
				return nil, "", fmt.Errorf("form file[%d]: FileName is required", i)
			}
			if f.Reader == nil {
				_ = w.Close()
				return nil, "", fmt.Errorf("form file[%d]: Reader is required", i)
			}

			ct := strings.TrimSpace(f.ContentType)
			if ct == "" {
				ext := strings.ToLower(filepath.Ext(f.FileName))
				ct = mime.TypeByExtension(ext)
				if ct == "" {
					ct = "application/octet-stream"
				}
			}

			// always create part with explicit content-type (ชัวร์สุด)
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes(f.FieldName),
					escapeQuotes(f.FileName),
				),
			)
			h.Set("Content-Type", ct)

			fw, err := w.CreatePart(h)
			if err != nil {
				_ = w.Close()
				return nil, "", fmt.Errorf("create part file[%d]: %w", i, err)
			}

			if _, err := io.Copy(fw, f.Reader); err != nil {
				_ = w.Close()
				return nil, "", fmt.Errorf("copy file[%d]: %w", i, err)
			}
		}

		if err := w.Close(); err != nil {
			return nil, "", fmt.Errorf("close multipart writer: %w", err)
		}

		return bytes.NewReader(buf.Bytes()), w.FormDataContentType(), nil
	}

	// JSON
	if opt.JSON != nil {
		b, err := json.Marshal(opt.JSON)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), "application/json", nil
	}

	return nil, "", nil
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
