package location

import (
	"regexp"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
)

// Denver's 78 official statistical neighborhoods, grouped the same way as
// https://en.wikipedia.org/wiki/List_of_neighborhoods_in_Denver. Used as a
// fallback when a project names a neighborhood but no specific street.
//
// A few entries carry more than one alias: either because the official name
// itself joins two names with a slash (e.g. "Gateway/Green Valley Ranch",
// where DOTI text typically uses just one side), or to deliberately EXCLUDE a
// risky generic alias — "South Platte" is dropped from "College View/South
// Platte" because it collides with the unrelated South Platte River, which
// shows up often elsewhere on the page. "Cherry Creek" is dropped entirely
// for the same reason: on the DOTI page every single mention is the Cherry
// Creek waterway/trail, never the upscale shopping-district neighborhood of
// the same name.
type neighborhoodEntry struct {
	canonical string
	aliases   []string
}

var neighborhoods = []neighborhoodEntry{
	// Central
	{"Baker", []string{"Baker"}},
	{"Capitol Hill", []string{"Capitol Hill"}},
	{"Central Business District", []string{"Central Business District"}},
	{"Cheesman Park", []string{"Cheesman Park"}},
	{"Cherry Creek", nil},
	{"City Park", []string{"City Park"}},
	{"City Park West", []string{"City Park West"}},
	{"Civic Center", []string{"Civic Center"}},
	{"Congress Park", []string{"Congress Park"}},
	{"Country Club", []string{"Country Club"}},
	{"Lincoln Park", []string{"Lincoln Park"}},
	{"North Capitol Hill", []string{"North Capitol Hill"}},
	{"Speer", []string{"Speer"}},
	{"Union Station", []string{"Union Station"}},
	// East
	{"Belcaro", []string{"Belcaro"}},
	{"Cory-Merrill", []string{"Cory-Merrill"}},
	{"East Colfax", []string{"East Colfax"}},
	{"Hale", []string{"Hale"}},
	{"Hilltop", []string{"Hilltop"}},
	{"Indian Creek", []string{"Indian Creek"}},
	{"Lowry", []string{"Lowry"}},
	{"Montclair", []string{"Montclair"}},
	{"Park Hill", []string{"Park Hill"}},
	{"Virginia Village", []string{"Virginia Village"}},
	{"Washington Virginia Vale", []string{"Washington Virginia Vale"}},
	{"Windsor", []string{"Windsor"}},
	// North
	{"Clayton", []string{"Clayton"}},
	{"Cole", []string{"Cole"}},
	{"Elyria-Swansea", []string{"Elyria-Swansea", "Elyria Swansea"}},
	{"Five Points", []string{"Five Points"}},
	{"Globeville", []string{"Globeville"}},
	{"North Park Hill", []string{"North Park Hill"}},
	{"Skyland", []string{"Skyland"}},
	{"South Park Hill", []string{"South Park Hill"}},
	{"Whittier", []string{"Whittier"}},
	// Northeast
	{"Central Park", []string{"Central Park"}},
	{"Denver International Airport", []string{"Denver International Airport"}},
	{"Gateway/Green Valley Ranch", []string{"Gateway", "Green Valley Ranch"}},
	{"Montbello", []string{"Montbello"}},
	{"Northeast Park Hill", []string{"Northeast Park Hill"}},
	// Northwest
	{"Auraria", []string{"Auraria"}},
	{"Berkeley", []string{"Berkeley"}},
	{"Chaffee Park", []string{"Chaffee Park"}},
	{"Highland", []string{"Highland"}},
	{"Jefferson Park", []string{"Jefferson Park"}},
	{"Regis", []string{"Regis"}},
	{"Sloan Lake", []string{"Sloan Lake"}},
	{"Sunnyside", []string{"Sunnyside"}},
	{"West Highland", []string{"West Highland"}},
	// South
	{"College View/South Platte", []string{"College View/South Platte", "College View"}},
	{"Overland", []string{"Overland"}},
	{"Platt Park", []string{"Platt Park"}},
	{"Rosedale", []string{"Rosedale"}},
	{"University", []string{"University"}},
	{"University Hills", []string{"University Hills"}},
	{"University Park", []string{"University Park"}},
	{"Washington Park", []string{"Washington Park"}},
	{"Washington Park West", []string{"Washington Park West"}},
	{"Wellshire", []string{"Wellshire"}},
	// Southeast
	{"Goldsmith", []string{"Goldsmith"}},
	{"Hampden", []string{"Hampden"}},
	{"Hampden South", []string{"Hampden South"}},
	{"Kennedy", []string{"Kennedy"}},
	{"Southmoor Park", []string{"Southmoor Park"}},
	// Southwest
	{"Bear Valley", []string{"Bear Valley"}},
	{"Fort Logan", []string{"Fort Logan"}},
	{"Harvey Park", []string{"Harvey Park"}},
	{"Harvey Park South", []string{"Harvey Park South"}},
	{"Marston", []string{"Marston"}},
	// West
	{"Athmar Park", []string{"Athmar Park"}},
	{"Barnum", []string{"Barnum"}},
	{"Barnum West", []string{"Barnum West"}},
	{"Mar Lee", []string{"Mar Lee"}},
	{"Ruby Hill", []string{"Ruby Hill"}},
	{"Sun Valley", []string{"Sun Valley"}},
	{"Valverde", []string{"Valverde"}},
	{"Villa Park", []string{"Villa Park"}},
	{"West Colfax", []string{"West Colfax"}},
	{"Westwood", []string{"Westwood"}},
}

// A couple of generic single-word names need a bit more care than a plain
// alias match — "University" alone is very often actually part of a longer
// proper name that isn't this neighborhood ("University Blvd", "University
// of Denver", "University Safer Streets Program"...). Rather than enumerate
// every such phrase, require that "University" NOT be immediately followed
// by another capitalized word or "of" — real mentions of the neighborhood
// itself are normally followed by a lowercase word ("the University area")
// or nothing. (Other overlaps, like "Speer" the street vs. "Speer" the
// neighborhood, are tolerated: this matcher only runs once every
// street-based pattern has already failed, which rules out most of those
// cases before we ever get here.)
var customPatterns = map[string]string{
	"University": `University(?!\s+(?:[A-Z]|of\b))`,
}

type aliasEntry struct {
	canonical string
	alias     string
	pattern   string
}

// aliasTable is sorted longest alias first, so e.g. "North Park Hill" wins
// over "Park Hill" at the same starting position, and once matched, the scan
// moves past it entirely rather than separately matching "Park Hill" again
// inside it.
var aliasTable = func() []aliasEntry {
	var table []aliasEntry
	for _, n := range neighborhoods {
		for _, alias := range n.aliases {
			pattern, ok := customPatterns[alias]
			if !ok {
				pattern = regexp.QuoteMeta(alias)
			}
			table = append(table, aliasEntry{n.canonical, alias, pattern})
		}
	}
	sort.SliceStable(table, func(i, j int) bool { return len(table[i].alias) > len(table[j].alias) })
	return table
}()

var neighborhoodRE = func() *regexp2.Regexp {
	patterns := make([]string, len(aliasTable))
	for i, e := range aliasTable {
		patterns[i] = e.pattern
	}
	return regexp2.MustCompile(`\b(?:`+strings.Join(patterns, "|")+`)\b`, regexp2.None)
}()

// findNeighborhoods finds every official Denver neighborhood named in text,
// returning their canonical names (deduped, in order of first mention).
// Matching requires the capitalized proper-noun form, same convention as the
// street patterns in extract.go — it's what keeps "the project area" from
// matching anything.
func findNeighborhoods(text string) []string {
	var found []string
	for _, match := range findAll(neighborhoodRE, text) {
		canonical := match[0]
		for _, e := range aliasTable {
			if e.alias == match[0] {
				canonical = e.canonical
				break
			}
		}
		if !contains(found, canonical) {
			found = append(found, canonical)
		}
	}
	return found
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
