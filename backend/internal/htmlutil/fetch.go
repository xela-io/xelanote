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
	MaxTextChars     = 50000
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
	metaTagRe                = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	imgTagRe                 = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	attrRe                   = regexp.MustCompile(`(?is)([a-zA-Z_:][a-zA-Z0-9_:\-]*)\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'>]+))`)
)

// FetchAndStripHTML fetches an HTML page and returns plain text for LLM prompts.
func FetchAndStripHTML(ctx context.Context, rawURL string) (string, error) {
	htmlBody, _, err := FetchHTML(ctx, rawURL)
	if err != nil {
		return "", err
	}

	text := StripHTML(htmlBody)
	if len(text) > MaxTextChars {
		text = text[:MaxTextChars]
	}
	return text, nil
}

// FetchHTML fetches an HTML page and returns raw HTML and the final URL after redirects.
func FetchHTML(ctx context.Context, rawURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", fmt.Errorf("%w: parse: %v", ErrInvalidURL, err)
	}

	if _, err := validateURL(parsed); err != nil {
		return "", "", err
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
		return "", "", fmt.Errorf("%w: request: %v", ErrFetchFailed, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", "", fmt.Errorf("%w: upstream returned status %d", ErrFetchFailed, resp.StatusCode)
	}

	if ct := strings.ToLower(resp.Header.Get("Content-Type")); ct != "" && !strings.Contains(ct, "text/html") {
		return "", "", fmt.Errorf("%w: unsupported content type", ErrFetchFailed)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("%w: read response: %v", ErrFetchFailed, err)
	}
	if len(body) > maxResponseBytes {
		return "", "", fmt.Errorf("%w: response too large", ErrFetchFailed)
	}

	finalURL := parsed.String()
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	return string(body), finalURL, nil
}

// FetchImage fetches an image from a public URL with SSRF protections.
func FetchImage(ctx context.Context, rawURL string, maxBytes int64) ([]byte, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, "", fmt.Errorf("%w: parse: %v", ErrInvalidURL, err)
	}

	if _, err := validateURL(parsed); err != nil {
		return nil, "", err
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
		return nil, "", fmt.Errorf("%w: request: %v", ErrFetchFailed, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("%w: upstream returned status %d", ErrFetchFailed, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: read response: %v", ErrFetchFailed, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", fmt.Errorf("%w: response too large", ErrFetchFailed)
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("%w: empty response body", ErrFetchFailed)
	}

	contentType := strings.ToLower(http.DetectContentType(body))
	switch {
	case strings.HasPrefix(contentType, "image/jpeg"):
		contentType = "image/jpeg"
	case strings.HasPrefix(contentType, "image/png"):
		contentType = "image/png"
	case strings.HasPrefix(contentType, "image/gif"):
		contentType = "image/gif"
	case strings.HasPrefix(contentType, "image/webp"):
		contentType = "image/webp"
	default:
		return nil, "", fmt.Errorf("%w: unsupported content type", ErrFetchFailed)
	}

	return body, contentType, nil
}

// ExtractImageCandidates extracts absolute image URLs from raw HTML.
func ExtractImageCandidates(rawHTML string, baseURL string) []string {
	if strings.TrimSpace(rawHTML) == "" || strings.TrimSpace(baseURL) == "" {
		return []string{}
	}

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return []string{}
	}

	candidates := make([]string, 0, 12)
	seen := make(map[string]bool)

	addCandidate := func(raw string) {
		raw = strings.TrimSpace(html.UnescapeString(raw))
		if raw == "" {
			return
		}
		if strings.HasPrefix(strings.ToLower(raw), "data:") {
			return
		}

		parsed, err := url.Parse(raw)
		if err != nil {
			return
		}
		abs := baseParsed.ResolveReference(parsed)
		if abs.Scheme != "http" && abs.Scheme != "https" {
			return
		}
		final := abs.String()
		if seen[final] {
			return
		}
		seen[final] = true
		candidates = append(candidates, final)
	}

	metaTags := metaTagRe.FindAllString(rawHTML, -1)
	for _, tag := range metaTags {
		attrs := parseTagAttributes(tag)
		prop := strings.ToLower(attrs["property"])
		name := strings.ToLower(attrs["name"])
		content := attrs["content"]
		if content == "" {
			continue
		}
		if prop == "og:image" || prop == "og:image:url" || name == "twitter:image" || name == "twitter:image:src" {
			addCandidate(content)
		}
	}

	imgTags := imgTagRe.FindAllString(rawHTML, -1)
	for _, tag := range imgTags {
		attrs := parseTagAttributes(tag)
		src := attrs["src"]
		if src != "" {
			addCandidate(src)
		}
		srcSet := attrs["srcset"]
		if srcSet != "" {
			first := strings.TrimSpace(strings.Split(srcSet, ",")[0])
			first = strings.TrimSpace(strings.Split(first, " ")[0])
			addCandidate(first)
		}
	}

	return candidates
}

func parseTagAttributes(tag string) map[string]string {
	attrs := make(map[string]string)
	matches := attrRe.FindAllStringSubmatch(tag, -1)
	for _, m := range matches {
		if len(m) < 6 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(m[1]))
		value := ""
		switch {
		case m[3] != "":
			value = m[3]
		case m[4] != "":
			value = m[4]
		default:
			value = m[5]
		}
		attrs[key] = strings.TrimSpace(value)
	}
	return attrs
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
