package kulon2

import (
	"fmt"
	stdhtml "html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/ruyam/kamps/internal/model"
)

var (
	sectionOpenPattern      = regexp.MustCompile(`(?is)<li\b[^>]*\bid=["']section-(\d+)["'][^>]*>`)
	sectionNamePattern      = regexp.MustCompile(`(?is)<h[23]\b[^>]*class=["'][^"']*\bsectionname\b[^"']*["'][^>]*>(.*?)</h[23]\s*>`)
	activityPattern         = regexp.MustCompile(`(?is)<li\b[^>]*class=["'][^"']*\bactivity\b[^"']*["'][^>]*>.*?</li\s*>`)
	moduleTypePattern       = regexp.MustCompile(`\bmodtype_([a-z]+)\b`)
	viewLinkPattern         = regexp.MustCompile(`(?i)href=["']([^"']*/(mod/[a-z]+)/view\.php\?[^"']*\bid=(\d+)[^"']*)["']`)
	instancePattern         = regexp.MustCompile(`(?is)class=["'][^"']*\binstancename\b[^"']*["'][^>]*>(.*?)</(?:a|span)\s*>`)
	datePairPattern         = regexp.MustCompile(`(?is)<strong[^>]*>([^<]+):</strong>\s*([^<]+)<`)
	attachmentPathPattern   = regexp.MustCompile(`/mod_assign/[a-z_]+/(\d+)/`)
	attachmentSuffixPattern = regexp.MustCompile(`/([^/]+)$`)
	submittedPattern        = regexp.MustCompile(`(?i)Submitted for grading`)
	notSubmittedPattern     = regexp.MustCompile(`(?i)Not submitted|No attempt`)
)

// Section is one course-page section; IDs remain LMS section IDs.
type Section struct{ ID, Name string }

// Module is a Moodle activity whose type and course-module ID were parsed from
// a course page. URL is the activity link exactly as found on the page;
// Section is the human-readable label and SectionID the LMS section id.
type Module struct{ Type, CMID, Name, Section, SectionID, URL string }

type Attachment struct{ ID, Filename, URL, ItemID string }

type AssignmentPage struct {
	DueRaw, DueISO, OpenRaw, LMSStatus string
	Attachments                        []Attachment
}

func ParseCoursePage(page []byte) ([]Section, []Module, error) {
	matches := sectionOpenPattern.FindAllSubmatchIndex(page, -1)
	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("course page contains no recognizable sections")
	}
	sections := make([]Section, 0, len(matches))
	modules := make([]Module, 0)
	for i, match := range matches {
		start := match[1]
		end := len(page)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		sectionID := string(page[match[2]:match[3]])
		body := page[start:end]
		section := Section{ID: sectionID}
		if name := sectionNamePattern.FindSubmatch(body); len(name) > 1 {
			section.Name = textContent(name[1])
		}
		sections = append(sections, section)
		label := section.Name
		if label == "" {
			label = sectionID
		}
		for _, block := range activityPattern.FindAll(body, -1) {
			typeMatch := moduleTypePattern.FindSubmatch(block)
			linkMatch := viewLinkPattern.FindSubmatch(block)
			if len(typeMatch) < 2 || len(linkMatch) < 4 {
				continue
			}
			name := "?"
			if instance := instancePattern.FindSubmatch(block); len(instance) > 1 {
				name = textContent(instance[1])
			}
			modules = append(modules, Module{Type: string(typeMatch[1]), CMID: string(linkMatch[3]), Name: name, Section: label, SectionID: sectionID, URL: string(linkMatch[1])})
		}
	}
	return sections, modules, nil
}

func ParseAssignmentPage(page []byte, cmid string) (AssignmentPage, error) {
	result := AssignmentPage{LMSStatus: "unknown", Attachments: []Attachment{}}
	if region := findDataRegion(page, "activity-dates"); len(region) > 0 {
		for _, pair := range datePairPattern.FindAllSubmatch(region, -1) {
			label := strings.ToLower(strings.TrimSpace(string(pair[1])))
			value := stdhtml.UnescapeString(strings.TrimSpace(string(pair[2])))
			switch label {
			case "due", "due date":
				if result.DueRaw == "" {
					result.DueRaw = value
				}
			case "opened", "available from":
				if result.OpenRaw == "" {
					result.OpenRaw = value
				}
			}
		}
	}
	if result.DueRaw != "" {
		for _, layout := range []string{"Monday, 2 January 2006, 3:04 PM", "Monday, 2 January 2006, 15:04", "2 January 2006, 15:04"} {
			if parsed, err := time.Parse(layout, result.DueRaw); err == nil {
				result.DueISO = parsed.Format("2006-01-02T15:04:05")
				break
			}
		}
	}
	if submittedPattern.Match(page) {
		result.LMSStatus = "submitted"
	} else if notSubmittedPattern.Match(page) {
		result.LMSStatus = "not_submitted"
	}
	result.Attachments = parseFileAnchors(page, cmid, true)
	return result, nil
}

// ParseFolderPage extracts the individual files behind a folder module.
func ParseFolderPage(page []byte, cmid string) []Attachment {
	return parseFileAnchors(page, cmid, false)
}

func parseFileAnchors(page []byte, cmid string, requireAssign bool) []Attachment {
	seenNames := map[string]int{}
	seenHrefs := map[string]bool{}
	out := []Attachment{}
	for _, anchor := range findElements(page, "a") {
		href := attribute(anchor, "href")
		if !strings.Contains(href, "pluginfile.php") {
			continue
		}
		if requireAssign && !strings.Contains(href, "/mod_assign/") {
			continue
		}
		rawURL := strings.ReplaceAll(href, "&amp;", "&")
		if seenHrefs[rawURL] {
			continue
		}
		seenHrefs[rawURL] = true
		filename := ""
		if filenameNode := findElement(anchor, "span", "filename"); filenameNode != nil {
			filename = strings.TrimSpace(textNode(filenameNode))
		}
		if filename == "" {
			if suffix := attachmentSuffixPattern.FindStringSubmatch(strings.SplitN(href, "?", 2)[0]); len(suffix) > 1 {
				filename = stdhtml.UnescapeString(suffix[1])
			}
		}
		if filename == "" {
			continue
		}
		seenNames[filename]++
		id := "M" + cmid + ":" + filename
		if seenNames[filename] > 1 {
			id += ":" + strconv.Itoa(seenNames[filename])
		}
		itemID := ""
		if item := attachmentPathPattern.FindStringSubmatch(href); len(item) > 1 {
			itemID = item[1]
		}
		out = append(out, Attachment{ID: id, Filename: filename, URL: rawURL, ItemID: itemID})
	}
	return out
}

func findDataRegion(page []byte, regionName string) []byte {
	document, err := html.Parse(strings.NewReader(string(page)))
	if err != nil {
		return nil
	}
	var found []byte
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				if attr.Key == "data-region" && attr.Val == regionName {
					found = []byte(textAndMarkup(node))
					return
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return found
}

func textAndMarkup(node *html.Node) string {
	var out strings.Builder
	var render func(*html.Node)
	render = func(current *html.Node) {
		if current.Type == html.TextNode {
			out.WriteString(stdhtml.EscapeString(current.Data))
			return
		}
		if current.Type == html.ElementNode {
			out.WriteByte('<')
			out.WriteString(current.Data)
			out.WriteByte('>')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			render(child)
		}
		if current.Type == html.ElementNode {
			out.WriteString("</")
			out.WriteString(current.Data)
			out.WriteByte('>')
		}
	}
	render(node)
	return out.String()
}

func findElements(page []byte, tagName string) []*html.Node {
	document, err := html.Parse(strings.NewReader(string(page)))
	if err != nil {
		return nil
	}
	return findElementsIn(document, tagName)
}

func findElementsIn(root *html.Node, tagName string) []*html.Node {
	var found []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tagName {
			found = append(found, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return found
}

func findElement(node *html.Node, tagName, className string) *html.Node {
	var walk func(*html.Node) *html.Node
	walk = func(current *html.Node) *html.Node {
		if current.Type == html.ElementNode && current.Data == tagName {
			for _, attr := range current.Attr {
				if attr.Key == "class" && hasClass(attr.Val, className) {
					return current
				}
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			if found := walk(child); found != nil {
				return found
			}
		}
		return nil
	}
	return walk(node)
}

func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func hasClass(classes, target string) bool {
	for _, className := range strings.Fields(classes) {
		if className == target {
			return true
		}
	}
	return false
}

func textNode(node *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			text.WriteString(current.Data)
			text.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(stdhtml.UnescapeString(text.String())), " ")
}

func textContent(raw []byte) string {
	document, err := html.Parse(strings.NewReader(string(raw)))
	if err != nil {
		return ""
	}
	return textNode(document)
}

func AssignmentID(cmid string) string { return "K" + cmid }
func MaterialID(cmid string) string   { return "M" + cmid }

func MapModule(course model.Course, module Module, siteURL string) (assignment *model.Assignment, material *model.Material) {
	switch module.Type {
	case "assign", "quiz":
		return &model.Assignment{ID: AssignmentID(module.CMID), CourseID: course.ID, CourseCode: course.Short, Name: module.Name, Section: module.Section}, nil
	case "resource":
		return nil, &model.Material{ID: MaterialID(module.CMID), CourseID: course.ID, CourseCode: course.Short, Name: module.Name, Section: module.Section, Kind: "file", URL: ResolveURL(siteURL, module.URL)}
	case "folder":
		// Folder children are materialized from the folder page at load time.
		return nil, nil
	case "url":
		return nil, &model.Material{ID: MaterialID(module.CMID), CourseID: course.ID, CourseCode: course.Short, Name: module.Name, Section: module.Section, Kind: "url", URL: ResolveURL(siteURL, module.URL)}
	case "book":
		return nil, &model.Material{ID: MaterialID(module.CMID), CourseID: course.ID, CourseCode: course.Short, Name: module.Name, Section: module.Section, Kind: "book", URL: ResolveURL(siteURL, module.URL)}
	default:
		return nil, nil
	}
}

// ResolveURL resolves a course-page link against the LMS base URL.
func ResolveURL(siteURL, raw string) string {
	if raw == "" {
		return ""
	}
	base, err := url.Parse(strings.TrimRight(siteURL, "/") + "/")
	if err != nil {
		return ""
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}
