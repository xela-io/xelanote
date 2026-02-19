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

func TestExtractImageCandidates(t *testing.T) {
	rawHTML := `
<html>
  <head>
    <meta property="og:image" content="/assets/hero.jpg" />
    <meta name="twitter:image" content="https://cdn.example.com/social.png" />
  </head>
  <body>
    <img src="/img/step1.png" />
    <img srcset="https://cdn.example.com/large.webp 2x, https://cdn.example.com/small.webp 1x" />
    <img src="data:image/png;base64,abc" />
  </body>
</html>`

	candidates := ExtractImageCandidates(rawHTML, "https://example.com/recipes/pasta")
	if len(candidates) < 4 {
		t.Fatalf("expected at least 4 candidates, got %d: %#v", len(candidates), candidates)
	}

	expected := []string{
		"https://example.com/assets/hero.jpg",
		"https://cdn.example.com/social.png",
		"https://example.com/img/step1.png",
		"https://cdn.example.com/large.webp",
	}

	for _, want := range expected {
		found := false
		for _, got := range candidates {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected candidate %q not found in %#v", want, candidates)
		}
	}
}
