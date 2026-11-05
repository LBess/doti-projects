package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/LBess/doti-projects/server/internal/project"
)

const (
	// LakewoodURL is the City of Lakewood's Cone Zone page, which lists its
	// current Public Works construction projects.
	LakewoodURL = "https://www.lakewoodco.gov/Local-Government/Departments/Public-Works/Cone-Zone"

	// lakewoodItemSelector picks the project cards out of the page's project
	// list. Each card is an <article> wrapping a link to the project's page,
	// with an <h3> title and a <p> description. Scoping to the list keeps
	// cards elsewhere on the page (quick links, the footer) out.
	lakewoodItemSelector  = ".oc-quick-list-stream article"
	lakewoodTitleSelector = "h3.list-item-title"
)

// LakewoodFetcher scrapes Lakewood's Cone Zone page. The list already carries
// each project's description, so one request is enough.
type LakewoodFetcher struct {
	URL    string
	Client *http.Client
}

func NewLakewoodFetcher(url string) *LakewoodFetcher {
	return &LakewoodFetcher{URL: url, Client: &http.Client{Timeout: fetchTimeout}}
}

// Fetch downloads the Cone Zone page and returns the projects on it.
func (fetcher *LakewoodFetcher) Fetch(ctx context.Context) ([]project.Project, error) {
	body, err := get(ctx, fetcher.Client, fetcher.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch Lakewood Cone Zone page: %w", err)
	}
	defer body.Close()
	return ParseLakewood(body, fetcher.URL)
}

// ParseLakewood parses the Cone Zone page. Links are resolved against
// pageURL and stripped of their query strings.
func ParseLakewood(r io.Reader, pageURL string) ([]project.Project, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse Lakewood Cone Zone page: %w", err)
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}

	var projects []project.Project
	seen := map[string]bool{}
	doc.Find(lakewoodItemSelector).Each(func(_ int, article *goquery.Selection) {
		title := joinText([]string{article.Find(lakewoodTitleSelector).First().Text()})
		if title == "" {
			return
		}

		var detailURL *string
		if href, ok := article.Find("a").First().Attr("href"); ok && strings.TrimSpace(href) != "" {
			if link, err := base.Parse(strings.TrimSpace(href)); err == nil {
				link.RawQuery = ""
				resolved := link.String()
				detailURL = &resolved
			}
		}

		key := sourceKey(title, detailURL)
		if seen[key] {
			return
		}
		seen[key] = true

		projects = append(projects, project.Project{
			SourceKey:   key,
			Title:       title,
			Description: joinText([]string{article.Find("p").First().Text()}),
			DetailURL:   detailURL,
		})
	})
	return projects, nil
}

// joinText joins pieces of text with single spaces, collapsing all other
// whitespace and dropping empty pieces.
func joinText(parts []string) string {
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}
