// Package scraper fetches each city's projects page(s) and parses them into
// projects. Denver's DOTI page is handled here; other cities get their own
// file.
package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/LBess/doti-projects/server/internal/project"
)

const (
	// DefaultURL is the DOTI Projects page.
	DefaultURL = "https://www.denvergov.org/Government/Agencies-Departments-Offices/Agencies-Departments-Offices-Directory/Department-of-Transportation-and-Infrastructure/Programs-Services/Projects"

	// Category is the only page section we keep. The page groups projects under
	// <h2> headings; the others (Public Infrastructure, Storm/Sanitary Sewer
	// Infrastructure) are skipped for now.
	Category = "Transportation/Mobility"

	userAgent    = "doti-projects-scraper/0.1"
	fetchTimeout = 30 * time.Second
	maxBodyBytes = 10 << 20

	siteOrigin     = "https://www.denvergov.org"
	panelSelector  = ".oc-wysiwyg-container-panel"
	detailSelector = "a.styled-button-calltoaction"
)

// Fetcher downloads and parses the DOTI projects page.
type Fetcher struct {
	URL    string
	Client *http.Client
}

func NewFetcher(url string) *Fetcher {
	return &Fetcher{URL: url, Client: &http.Client{Timeout: fetchTimeout}}
}

// Fetch downloads the page and returns its Transportation/Mobility projects.
func (fetcher *Fetcher) Fetch(ctx context.Context) ([]project.Project, error) {
	body, err := get(ctx, fetcher.Client, fetcher.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch DOTI page: %w", err)
	}
	defer body.Close()
	return Parse(body)
}

// get requests url and returns its body, capped at maxBodyBytes. The caller
// closes it.
func get(ctx context.Context, client *http.Client, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(resp.Body, maxBodyBytes), resp.Body}, nil
}

// Parse parses the DOTI "Projects" page HTML. The page groups projects as
// <h2> category headers followed by .oc-wysiwyg-container-panel blocks, each
// containing an <h3> title and a .oc-wysiwyg-container-panel-content
// description with an optional "Read more" link. Only projects under the
// Category heading are returned.
func Parse(reader io.Reader) ([]project.Project, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("parse DOTI page: %w", err)
	}

	var projects []project.Project
	seen := map[string]bool{}
	currentCategory := ""

	// Walk the page in document order so each panel is attributed to the <h2>
	// that most recently preceded it. The page has no <main>, so search the
	// whole document.
	doc.Find("h2, " + panelSelector).Each(func(_ int, el *goquery.Selection) {
		if goquery.NodeName(el) == "h2" {
			// Some panels embed their own <h2>s ("Background Information",
			// "Contact", ...). Those aren't category headings.
			if el.ParentsFiltered(panelSelector).Length() > 0 || el.HasClass("visuallyhidden") {
				return
			}
			currentCategory = strings.TrimSpace(el.Text())
			return
		}
		if currentCategory != Category {
			return
		}

		title := strings.TrimSpace(el.Find("h3").First().Text())
		content := el.Find(".oc-wysiwyg-container-panel-content").First()
		if title == "" || content.Length() == 0 {
			return
		}

		var detailURL *string
		if href, ok := content.Find(detailSelector).First().Attr("href"); ok && strings.TrimSpace(href) != "" {
			resolved := resolveDetailURL(strings.TrimSpace(href))
			detailURL = &resolved
		}

		key := sourceKey(title, detailURL)
		if seen[key] {
			return
		}
		seen[key] = true

		projects = append(projects, project.Project{
			SourceKey:   key,
			Title:       title,
			Description: description(content),
			DetailURL:   detailURL,
		})
	})

	return projects, nil
}

func description(content *goquery.Selection) string {
	var parts []string
	content.Find("p, li").Each(func(_ int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())
		if text != "" && !strings.HasPrefix(text, "Read more about") {
			parts = append(parts, text)
		}
	})
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

func resolveDetailURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	return siteOrigin + href
}

// sourceKey derives a project's stable identity. The page position is not
// usable (it shifts whenever DOTI adds or reorders a project), so:
//
//   - With a detail URL, the key is the URL minus everything that churns: the
//     scheme, a leading "www.", the query string (links carry a rotating
//     ?transfer=<guid>), and, for denvergov.org, the directory prefix before
//     "Programs-Services/" (the same project is linked as both
//     /Government/Departments/... and
//     /Government/Agencies-Departments-Offices/...), and for lakewoodco.gov,
//     "Local-Government" vs "Government". The fragment is kept,
//     since distinct projects link to different #sections of one page.
//   - Without one, the key is the normalized title.
func sourceKey(title string, detailURL *string) string {
	if detailURL != nil {
		if u, err := url.Parse(*detailURL); err == nil && u.Host != "" {
			host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
			path := strings.TrimSuffix(strings.ToLower(u.Path), "/")
			if host == "denvergov.org" {
				if _, rest, ok := strings.Cut(path, "/programs-services/"); ok {
					path = "programs-services/" + rest
				}
			}
			if host == "lakewoodco.gov" {
				// The same page is linked under both /Local-Government/ and
				// /Government/.
				path = strings.Replace(path, "/local-government/", "/government/", 1)
			}
			key := host + "/" + strings.TrimPrefix(path, "/")
			if u.Fragment != "" {
				key += "#" + strings.ToLower(u.Fragment)
			}
			return "url:" + key
		}
	}
	return "title:" + strings.Join(strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}), "-")
}
