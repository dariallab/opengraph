// Package opengraph implements and parses "The Open Graph Protocol" of web pages.
// See http://ogp.me/ for more information.
package opengraph

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

const (
	// HTMLMetaTag is a tag name of <meta>
	HTMLMetaTag string = "meta"
	// HTMLLinkTag is a tag name of <link>
	HTMLLinkTag string = "link"
	// HTMLTitleTag is a tag name of <title>
	HTMLTitleTag string = "title"
)

// OpenGraph represents web page information according to OGP <ogp.me>,
// and some more additional informations like URL.Host and so.
type OpenGraph struct {

	// Basic Metadata
	// https://ogp.me/#metadata
	Title string  `json:"title"`
	Type  string  `json:"type"`
	Image []Image `json:"image"` // could be multiple
	URL   string  `json:"url"`

	// Optional Metadata
	// https://ogp.me/#optional
	Audio       []Audio  `json:"audio"` // could be multiple
	Description string   `json:"description"`
	Determiner  string   `json:"determiner"` // TODO: enum of (a, an, the, "", auto)
	Locale      string   `json:"locale"`
	LocaleAlt   []string `json:"locale_alternate"`
	SiteName    string   `json:"site_name"`
	Video       []Video  `json:"video"`

	// Additional (unofficial)
	Favicon Favicon `json:"favicon"`

	// Intent represents how to fetch, parse, and complete properties
	// of this OpenGraph object.
	// This SHOULD NOT have any meaning for "OpenGraph Protocol".
	Intent Intent `json:"-"`
}

// New ...
func New(rawurl string) *OpenGraph {
	return &OpenGraph{Intent: Intent{URL: rawurl}}
}

// Fetch creates and parses OpenGraph with specified URL.
func Fetch(rawurl string) (*OpenGraph, error) {
	return FetchWithContext(context.Background(), rawurl)
}

// FetchWithContext creates and parses OpenGraph with specified URL.
// Timeout can be handled with provided context.
func FetchWithContext(ctx context.Context, rawurl string) (*OpenGraph, error) {
	og := &OpenGraph{Intent: Intent{URL: rawurl}}
	err := og.Fetch(ctx)
	return og, err
}

// Fetch ...
func (og *OpenGraph) Fetch(ctx context.Context) error {

	if og.Intent.URL == "" {
		return fmt.Errorf("no URL given yet")
	}

	if og.Intent.HTTPClient == nil {
		og.Intent.HTTPClient = http.DefaultClient
	}

	req, err := http.NewRequest("GET", og.Intent.URL, nil)
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	req = req.WithContext(ctx)

	res, err := og.Intent.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("Content type must be text/html")
	}

	if err = og.Parse(res.Body); err != nil {
		return err
	}

	return nil
}

// Parse parses http.Response.Body and construct OpenGraph informations.
// Caller should close body after it gets parsed.
func (og *OpenGraph) Parse(body io.Reader) error {
	node, err := html.Parse(body)
	if err != nil {
		return err
	}
	return og.walk(node)
}

func (og *OpenGraph) walk(node *html.Node) error {

	if node.Type == html.ElementNode {
		switch {
		case node.Data == HTMLMetaTag:
			return MetaTag(node).Contribute(og)
		case !og.Intent.Strict && node.Data == HTMLTitleTag:
			return TitleTag(node).Contribute(og)
		case !og.Intent.Strict && node.Data == HTMLLinkTag:
			return LinkTag(node).Contribute(og)
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		og.walk(child)
	}

	return nil
}
