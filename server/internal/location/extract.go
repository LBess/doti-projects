// Package location guesses a geocodable place for a DOTI project from its
// free-form title and description.
//
// The heuristics are regexes over prose, so they use regexp2 (a backtracking
// engine) rather than the standard library: the street patterns rely on
// lookaheads, which Go's RE2-based regexp doesn't support. Every pattern is
// deliberately case-sensitive — capitalization is what tells a real street
// name ("Downing St.") apart from ordinary lowercase prose.
package location

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/LBess/doti-projects/server/internal/project"
)

// Guess is a place worth geocoding. Each of Queries is geocoded and the pin
// goes at their centroid: a lone query for a point, the two ends of a
// corridor (so the pin sits at its midpoint), or every named neighborhood.
type Guess struct {
	Kind    string
	Queries []string
	Label   string
}

const citySuffix = "Denver, CO"

// Street-name suffix words, spelled out or abbreviated. Used to recognize a
// token sequence as a plausible street name rather than arbitrary prose. The
// trailing `(?![a-z])` is load-bearing: without it, the un-anchored "St"
// alternative matches as a prefix of unrelated words like "Storm" or "Stout"
// (regex has no implicit word-boundary at the end of an alternation).
const suffix = `(?:St(?:reet)?|Ave(?:nue)?|Blvd|Boulevard|Rd|Road|Dr(?:ive)?|Pkwy|Parkway|Way|Pl(?:ace)?|Ln|Lane|Ct|Court|Cir|Circle|Trl|Trail|Hwy|Highway)s?\.?(?![a-z])`

// A leading direction, e.g. "N", "S.", "East", "West" — only valid as a
// prefix, never as the whole name (so "East Avenues" alone can't match).
const direction = `(?:[NSEW]\.?|North|South|East|West)`

// A capitalized proper-noun word that ISN'T itself a bare direction word.
const nameWord = `(?!North\b|South\b|East\b|West\b)[A-Z][a-zA-Z']*`
const ordinal = `\d+(?:st|nd|rd|th)`
const core = `(?:` + nameWord + `|` + ordinal + `)`

// A handful of major Denver streets/highways that are commonly referenced
// but don't take any of the generic suffixes above (nobody writes "Broadway
// St." or "I-25 Rd."), so without this they'd never match street at all —
// which silently breaks any pattern where one of the two bounds is one of
// these (e.g. "...between Broadway and Colorado Boulevard").
const exceptionStreet = `Broadway|I-25|I-70|I-76|US[- ]?85`

// A street name: an optional direction prefix, 1-3 core words, then a
// recognized suffix (e.g. "W. 1st Avenue", "University Blvd."), or one of
// the suffix-less exceptions above.
const street = `(?:` + exceptionStreet + `|(?:` + direction + `\.?\s+)?` + core + `(?:\s+` + core + `){0,2}\s+` + suffix + `)`

// bound is one end of a "between A and B" range: a full street name, or a
// bare ordinal that borrows its suffix from the other end.
const bound = `(?:` + street + `|` + ordinal + `)`

// compass is the optional "to the South" after a "bounded by" street.
const compass = `(?:\s+to\s+the\s+(?:[Nn]orth|[Ss]outh|[Ee]ast|[Ww]est))?`

func compile(pattern string) *regexp2.Regexp {
	return regexp2.MustCompile(pattern, regexp2.None)
}

var (
	streetRE = compile(street)

	corridorFromToRE   = compile(`(` + street + `)\s+[Ff]rom\s+(` + street + `)\s+[Tt]o\s+(` + street + `)`)
	numberedRangeRE    = compile(`\b(` + ordinal + `)\s+(?:[Tt]o|[Aa]nd)\s+(` + ordinal + `)\s+(` + suffix + `)`)
	corridorBetweenRE  = compile(`(` + street + `)\s+between\s+(` + bound + `)\s+(?:[Aa]nd|&)\s+(` + bound + `)`)
	boundsOnlyBetweenR = compile(`between\s+(` + bound + `)\s+(?:[Aa]nd|&)\s+(` + bound + `)`)
	boundedByRE        = compile(`bounded\s+by\s+(` + street + `)` + compass + `\s*(?:,|[Aa]nd|&)\s*(?:and\s+)?(` + street + `)`)
	boundsOnlyRE       = compile(`(` + street + `)\s+[Tt]o\s+(` + street + `)`)
	intersectionRE     = compile(`(` + street + `)\s+(?:[Aa]nd|&)\s+(` + street + `)`)
	bridgeOverRE       = compile(`(` + street + `)\s+(?:[Bb]ridge\s+)?over\s+(` + street + `)`)

	bareOrdinalRE = regexp.MustCompile(`^` + ordinal + `\.?$`)
	laneSuffixRE  = regexp.MustCompile(`\s+(?:Lane|Ln)$`)
	whitespaceRE  = regexp.MustCompile(`\s+`)
	trailingDotRE = regexp.MustCompile(`\.$`)
	trailingSRE   = regexp.MustCompile(`s\.?$`)
)

// findAll returns the capture groups of every non-overlapping match, with
// the whole match at index 0.
func findAll(re *regexp2.Regexp, text string) [][]string {
	var matches [][]string
	m, _ := re.FindStringMatch(text)
	for m != nil {
		groups := m.Groups()
		match := make([]string, len(groups))
		for i, g := range groups {
			match[i] = g.String()
		}
		matches = append(matches, match)
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

// findFirst returns the capture groups of the first match, or nil.
func findFirst(re *regexp2.Regexp, text string) []string {
	m, _ := re.FindStringMatch(text)
	if m == nil {
		return nil
	}
	groups := m.Groups()
	match := make([]string, len(groups))
	for i, g := range groups {
		match[i] = g.String()
	}
	return match
}

func cleanStreet(raw string) string {
	s := whitespaceRE.ReplaceAllString(raw, " ")
	s = trailingDotRE.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func findStreets(text string) []string {
	var streets []string
	for _, m := range findAll(streetRE, text) {
		streets = append(streets, cleanStreet(m[0]))
	}
	return streets
}

// firstStreetOtherThan returns the first street named in the title that
// isn't one of the given bounds — the corridor for a bounds-only phrase.
func firstStreetOtherThan(title string, others ...string) (string, bool) {
	for _, s := range findStreets(title) {
		if !contains(others, s) {
			return s, true
		}
	}
	return "", false
}

// refineCorridor handles titles like "Alameda Lane Repurposing Demonstration
// Project", which use "Lane" as a project-type word, not a street suffix, so
// "Alameda Lane" can't be geocoded. When the corridor ends in "Lane", look
// for the same base name with a real street suffix elsewhere in the text
// ("the Alameda Avenue corridor") and use that instead. Genuine "...Lane"
// streets with no such alternative are left alone.
func refineCorridor(corridor, text string) string {
	cleaned := cleanStreet(corridor)
	if !laneSuffixRE.MatchString(cleaned) {
		return cleaned
	}
	base := laneSuffixRE.ReplaceAllString(cleaned, "")
	re := compile(`\b` + regexp.QuoteMeta(base) + `\s+(` + suffix + `)`)
	for _, m := range findAll(re, text) {
		if alt := cleanStreet(m[0]); !laneSuffixRE.MatchString(alt) {
			return alt
		}
	}
	return cleaned
}

func geocodeQuery(place, other string) string {
	if other != "" {
		return fmt.Sprintf("%s & %s, %s", place, other, citySuffix)
	}
	return fmt.Sprintf("%s, %s", place, citySuffix)
}

func isBareOrdinal(s string) bool {
	return bareOrdinalRE.MatchString(strings.TrimSpace(s))
}

func lastWord(s string) string {
	parts := strings.Fields(s)
	return parts[len(parts)-1]
}

// singularizeSuffix turns "Avenues"/"Ave." into the canonical singular
// "Avenue", for borrowing a suffix.
func singularizeSuffix(word string) string {
	return strings.TrimSuffix(strings.TrimSuffix(word, "."), "s")
}

// resolveBound handles a bound in a "between A and B" / "A to B" range, which
// is either a fully named street, or a bare ordinal ("6th") that borrows its
// suffix from the other bound — e.g. "Santa Fe Drive between 6th and Colfax
// Avenues" means 6th *Avenue*. other is that range partner, used only when raw
// is bare.
func resolveBound(raw, other string) string {
	cleaned := cleanStreet(raw)
	if !isBareOrdinal(cleaned) {
		return cleaned
	}
	return cleaned + " " + singularizeSuffix(lastWord(cleanStreet(other)))
}

func corridor(along, from, to string) *Guess {
	return &Guess{
		Kind:    project.KindCorridor,
		Queries: []string{geocodeQuery(from, along), geocodeQuery(to, along)},
		Label:   fmt.Sprintf("%s: %s to %s", along, from, to),
	}
}

// tryCorridorFromTo matches "<corridor> from <A> to <B>" — the single clearest
// signal, capturing all three streets in one pass (e.g. "redesign 1st Avenue
// from Downing St. to University Blvd.").
func tryCorridorFromTo(text string) *Guess {
	m := findFirst(corridorFromToRE, text)
	if m == nil {
		return nil
	}
	return corridor(cleanStreet(m[1]), cleanStreet(m[2]), cleanStreet(m[3]))
}

// tryNumberedRange handles the common shorthand where only the second bound
// spells out the suffix, e.g. "10th to 12th Avenues" or "6th and 13th
// Avenues" (both are numbered avenues).
func tryNumberedRange(text, title string) *Guess {
	m := findFirst(numberedRangeRE, text)
	if m == nil {
		return nil
	}
	suffixWord := trailingDotRE.ReplaceAllString(trailingSRE.ReplaceAllString(m[3], ""), "")
	from := m[1] + " " + suffixWord
	to := m[2] + " " + suffixWord

	// Guard against e.g. "17th and 18th Avenues Bus Priority..." — the only
	// "street" findable in that title IS "18th Avenues", one of our own
	// bounds, which would otherwise produce a nonsensical self-referential
	// corridor ("18th Avenues: 17th Avenue to 18th Avenue").
	var found string
	for _, s := range findStreets(title) {
		if !strings.EqualFold(s, from) && !strings.EqualFold(s, to) {
			found = s
			break
		}
	}
	if found == "" {
		return nil
	}
	return corridor(refineCorridor(found, text), from, to)
}

// tryCorridorBetween matches "<corridor> between <A> and <B>" — e.g. "Santa
// Fe Drive between 6th and Colfax Avenues". Either bound may be a bare
// ordinal borrowing the other's suffix (see resolveBound).
func tryCorridorBetween(text string) *Guess {
	m := findFirst(corridorBetweenRE, text)
	if m == nil {
		return nil
	}
	if isBareOrdinal(cleanStreet(m[2])) && isBareOrdinal(cleanStreet(m[3])) {
		return nil
	}
	return corridor(cleanStreet(m[1]), resolveBound(m[2], m[3]), resolveBound(m[3], m[2]))
}

// tryBoundsOnlyBetween matches "between <A> and <B>" without a leading
// corridor phrase — falls back to the first street mentioned in the title,
// same as tryBoundsOnly.
func tryBoundsOnlyBetween(text, title string) *Guess {
	m := findFirst(boundsOnlyBetweenR, text)
	if m == nil {
		return nil
	}
	if isBareOrdinal(cleanStreet(m[1])) && isBareOrdinal(cleanStreet(m[2])) {
		return nil
	}
	from := resolveBound(m[1], m[2])
	to := resolveBound(m[2], m[1])
	found, ok := firstStreetOtherThan(title, from, to)
	if !ok {
		return nil
	}
	return corridor(refineCorridor(found, text), from, to)
}

// tryBoundedBy matches "bounded by <A> [to the South] and <B> [to the North]"
// — e.g. "the Federal Blvd. corridor bounded by Floyd Ave. to the South and
// Columbine Rd. to the North". The corridor is the first other street named
// in the title.
func tryBoundedBy(text, title string) *Guess {
	m := findFirst(boundedByRE, text)
	if m == nil {
		return nil
	}
	from := cleanStreet(m[1])
	to := cleanStreet(m[2])
	found, ok := firstStreetOtherThan(title, from, to)
	if !ok {
		return nil
	}
	return corridor(refineCorridor(found, text), from, to)
}

// tryBoundsOnly matches "<A> to <B>" without a leading corridor phrase —
// falls back to the first street mentioned in the title as the corridor, as
// long as it's distinct from both bounds.
func tryBoundsOnly(text, title string) *Guess {
	m := findFirst(boundsOnlyRE, text)
	if m == nil {
		return nil
	}
	from := cleanStreet(m[1])
	to := cleanStreet(m[2])
	found, ok := firstStreetOtherThan(title, from, to)
	if !ok {
		return nil
	}
	return corridor(refineCorridor(found, text), from, to)
}

// tryIntersection matches "<A> and/& <B>" — a direct intersection, e.g.
// "Buchtel Boulevard & Evans Intersection Improvements". Tried on the title
// first (highest signal), then the description.
func tryIntersection(text string) *Guess {
	m := findFirst(intersectionRE, text)
	if m == nil {
		return nil
	}
	a := cleanStreet(m[1])
	b := cleanStreet(m[2])
	if strings.EqualFold(a, b) {
		return nil
	}
	return intersection(a, b)
}

// tryBridgeOver matches "<A> (Bridge) over <B>" — bridge/crossing projects,
// treated like an intersection between the two crossing features.
func tryBridgeOver(text string) *Guess {
	m := findFirst(bridgeOverRE, text)
	if m == nil {
		return nil
	}
	return intersection(cleanStreet(m[1]), cleanStreet(m[2]))
}

func intersection(a, b string) *Guess {
	return &Guess{
		Kind:    project.KindPoint,
		Queries: []string{geocodeQuery(a, b)},
		Label:   a + " & " + b,
	}
}

// tryNeighborhoods finds one or more named Denver neighborhoods, e.g.
// "Globeville Elyria-Swansea (GES)" — the pin is placed at the centroid of
// whichever ones are found. This is the last resort: it only runs once every
// street-based pattern above has already failed to find anything more
// specific.
func tryNeighborhoods(text string) *Guess {
	names := findNeighborhoods(text)
	if len(names) == 0 {
		return nil
	}
	queries := make([]string, len(names))
	for i, name := range names {
		queries[i] = geocodeQuery(name, "")
	}
	label := names[0] + " neighborhood"
	if len(names) > 1 {
		label = strings.Join(names, ", ") + " neighborhoods"
	}
	return &Guess{Kind: project.KindNeighborhood, Queries: queries, Label: label}
}

// Extract attempts to guess a geocodable location for a DOTI project from its
// title and description. It returns nil when no pattern confidently applies
// (city-wide plans, programs, studies with no single site, etc.) — those
// projects are surfaced as unmapped instead of getting a pin.
func Extract(title, description string) *Guess {
	text := title + ". " + description

	for _, try := range []func() *Guess{
		func() *Guess { return tryCorridorFromTo(text) },
		func() *Guess { return tryCorridorBetween(text) },
		func() *Guess { return tryNumberedRange(text, title) },
		func() *Guess { return tryBoundsOnlyBetween(text, title) },
		func() *Guess { return tryBoundedBy(text, title) },
		func() *Guess { return tryBoundsOnly(text, title) },
		func() *Guess { return tryIntersection(title) },
		func() *Guess { return tryIntersection(text) },
		func() *Guess { return tryBridgeOver(text) },
		func() *Guess { return tryNeighborhoods(text) },
	} {
		if guess := try(); guess != nil {
			return guess
		}
	}
	return nil
}
