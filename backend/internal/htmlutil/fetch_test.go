package htmlutil

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestValidateURLRejectsInvalidScheme(t *testing.T) {
	u, err := url.Parse("ftp://example.com/recipe")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	_, err = validateURL(u)
	if err == nil || !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("expected ErrInvalidURL, got: %v", err)
	}
}

func TestValidateURLRejectsPrivateAddress(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1/recipe")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	_, err = validateURL(u)
	if err == nil || !errors.Is(err, ErrDisallowedAddress) {
		t.Fatalf("expected ErrDisallowedAddress, got: %v", err)
	}
}

func TestStripHTMLRemovesScriptStyleAndTags(t *testing.T) {
	input := `
<html>
  <head>
    <style>.x{display:none}</style>
    <script>console.log("secret")</script>
  </head>
  <body>
    <nav>Menu</nav>
    <h1>Tomatensuppe</h1>
    <p>Zutaten: <strong>Tomaten</strong>, Zwiebeln</p>
  </body>
</html>`

	out := StripHTML(input)
	if strings.Contains(out, "console.log") {
		t.Fatalf("script content leaked: %q", out)
	}
	if strings.Contains(strings.ToLower(out), "menu") {
		t.Fatalf("nav content leaked: %q", out)
	}
	if !strings.Contains(out, "Tomatensuppe") || !strings.Contains(out, "Zutaten: Tomaten, Zwiebeln") {
		t.Fatalf("expected content missing: %q", out)
	}
}
