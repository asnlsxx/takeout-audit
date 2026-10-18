package client

import (
	"fmt"
	"net/http"
	"net/url"
)

type Option func(*http.Client, string) error

// WithAuditOnly constrains the transport, not just command code. API keys may
// need asset.upload permission even though this endpoint does not upload.
func WithAuditOnly() Option {
	return func(c *http.Client, server string) error {
		u, err := url.Parse(server)
		if err != nil || u == nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return fmt.Errorf("audit requires a server origin without credentials, path, query or fragment")
		}
		c.Transport = auditTransport{next: c.Transport, origin: u.Scheme + "://" + u.Host}
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		return nil
	}
}

type auditTransport struct {
	next   http.RoundTripper
	origin string
}

func (t auditTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	allowed := r.Method == http.MethodGet && (r.URL.Path == "/api/users/me" || r.URL.Path == "/api/server/version") ||
		r.Method == http.MethodPost && r.URL.Path == "/api/assets/bulk-upload-check"
	if !allowed || r.URL.Scheme+"://"+r.URL.Host != t.origin || r.URL.RawQuery != "" || r.URL.User != nil || r.URL.RawPath != "" {
		return nil, fmt.Errorf("audit transport blocked non-allowlisted request")
	}
	return t.next.RoundTrip(r)
}
