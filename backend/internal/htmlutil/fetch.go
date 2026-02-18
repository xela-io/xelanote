package htmlutil

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	maxResponseBytes = 2 << 20 // 2 MiB
	maxTextChars     = 50000
	maxRedirects     = 3
	requestTimeout   = 15 * time.Second
	userAgent        = "Xelanote/1.0 RecipeImporter"
)

var (
	ErrInvalidURL        = errors.New("invalid url")
	ErrDisallowedAddress = errors.New("disallowed url address")
	ErrFetchFailed       = errors.New("failed to fetch url")

	spaceRe                  = regexp.MustCompile(`\s+`)
	spaceBeforePunctuationRe = regexp.MustCompile(`\s+([,.;:!?])`)
	tagRe                    = regexp.MustCompile(`(?is)<[^>]+>`)
	scriptRe                 = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe                  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	navRe                    = regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`)
)

// FetchAndStripHTML fetches an HTML page and returns plain text for LLM prompts.
func FetchAndStripHTML(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("%w: parse: %v", ErrInvalidURL, err)
	}

	if _, err := validateURL(parsed); err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout:   requestTimeout,
		Transport: newPinnedTransport(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("%w: too many redirects", ErrFetchFailed)
			}
			if _, err := validateURL(req.URL); err != nil {
				return err
			}
			req.Header.Set("User-Agent", userAgent)
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("%w: request: %v", ErrFetchFailed, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("%w: upstream returned status %d", ErrFetchFailed, resp.StatusCode)
	}

	if ct := strings.ToLower(resp.Header.Get("Content-Type")); ct != "" && !strings.Contains(ct, "text/html") {
		return "", fmt.Errorf("%w: unsupported content type", ErrFetchFailed)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("%w: read response: %v", ErrFetchFailed, err)
	}
	if len(body) > maxResponseBytes {
		return "", fmt.Errorf("%w: response too large", ErrFetchFailed)
	}

	text := StripHTML(string(body))
	if len(text) > maxTextChars {
		text = text[:maxTextChars]
	}
	return text, nil
}

func validateURL(u *url.URL) ([]netip.Addr, error) {
	if u == nil || u.Host == "" {
		return nil, fmt.Errorf("%w: host required", ErrInvalidURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme must be http or https", ErrInvalidURL)
	}
	if u.User != nil {
		return nil, fmt.Errorf("%w: userinfo is not allowed", ErrInvalidURL)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("%w: host required", ErrInvalidURL)
	}

	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("%w: dns resolution failed", ErrFetchFailed)
	}

	allowed := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok {
			continue
		}
		if isBlockedIP(addr) {
			return nil, fmt.Errorf("%w: private/loopback/link-local addresses are not allowed", ErrDisallowedAddress)
		}
		allowed = append(allowed, addr)
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("%w: no usable ip addresses", ErrDisallowedAddress)
	}

	return allowed, nil
}

func isBlockedIP(addr netip.Addr) bool {
	return addr.IsPrivate() ||
		addr.IsLoopback() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}

func newPinnedTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			fresh, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("%w: dns lookup failed", ErrFetchFailed)
			}
			for _, candidate := range fresh {
				addr, ok := netip.AddrFromSlice(candidate.IP)
				if !ok || isBlockedIP(addr) {
					continue
				}
				// Dial directly to the resolved IP to pin DNS resolution per connection.
				return dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
			}

			return nil, fmt.Errorf("%w: no allowed ip", ErrDisallowedAddress)
		},
		DisableCompression: false,
	}
}

// StripHTML removes tags and keeps visible text only.
func StripHTML(input string) string {
	out := scriptRe.ReplaceAllString(input, " ")
	out = styleRe.ReplaceAllString(out, " ")
	out = navRe.ReplaceAllString(out, " ")
	out = tagRe.ReplaceAllString(out, " ")
	out = html.UnescapeString(out)
	out = strings.ReplaceAll(out, "\u00a0", " ")
	out = spaceRe.ReplaceAllString(out, " ")
	out = spaceBeforePunctuationRe.ReplaceAllString(out, "$1")
	return strings.TrimSpace(out)
}
