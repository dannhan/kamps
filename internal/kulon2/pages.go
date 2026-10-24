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
	sectionOpenPattern = regexp.MustCompile(`(?is)<li\b[^>]*\bid=["']section-(\d+)["'][^>]*>`)
	sectionNamePattern = regexp.MustCompile(`(?is)<h[23]\b[^>]*class=["'][^"']*\bsectionname\b[^"']*["'][^>]*>(.*?)</h[23]\s*>`)
	activityPattern    = regexp.MustCompile(`(?is)<li\b[^>]*class=["'][^"']*\bactivity\b[^"']*["'][^>]*>.*?</li\s*>`)
	moduleTypePattern  = regexp.MustCompile(`\bmodtype_([a-z]+)\b`)
	viewLinkPattern    = regexp.MustCompile(`(?i)href=["']([^"']*/(mod/[a-z]+)/view\.php\?[^"']*\bid=(\d+)[^"']*)["']`)
	instancePattern    = regexp.MustCompile(`(?is)class=["'][^"']*\binstancename\b[^"']*["'][^>]*>(.*?)</(?:a|span)\s*>`)
	activityDates      = regexp.MustCompile(`(?is)data-region=["']activity-dates["'](.*?)(?:<div class=["']activity-description|</section\s*>)`)
	datePairPattern    = regexp.MustCompile(`(?is)<strong[^>]*>([^<]+):</strong>\s*([^<]+)<`)
	attachmentPattern  = regexp.MustCompile(`(?i)href=["']([^"']*pluginfile\.php/[^"']*/mod_assign/[^"']*)["']`)
)

// Section is one course-page section; IDs remain LMS section IDs.
type Section struct{ ID, Name string }

// Module is a Moodle activity whose type and course-module ID were parsed from
// a course page.
type Module struct{ Type, CMID, Name, Section string }

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
			modules = append(modules, Module{Type: string(typeMatch[1]), CMID: string(linkMatch[3]), Name: name, Section: sectionID})
		}
	}
	return sections, modules, nil
}

func ParseAssignmentPage(page []byte, cmid string) (AssignmentPage, error) {
	result := AssignmentPage{LMSStatus: "unknown", Attachments: []Attachment{}}
	if block := activityDates.FindSubmatch(page); len(block) > 1 {
		for _, pair := range datePairPattern.FindAllSubmatch(block[1], -1) {
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
	if regexp.MustCompile(`(?i)Submitted for grading`).Match(page) {
		result.LMSStatus = "submitted"
	} else if regexp.MustCompile(`(?i)Not submitted|No attempt`).Match(page) {
		result.LMSStatus = "not_submitted"
	}
	seenNames := map[string]int{}
	for _, match := range attachmentPattern.FindAllSubmatch(page, -1) {
		rawURL := stdhtml.UnescapeString(strings.ReplaceAll(string(match[1]), "&amp;", "&"))
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return AssignmentPage{}, fmt.Errorf("parse attachment URL: %w", err)
		}
		pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		filename := ""
		if len(pathParts) > 0 {
			filename, _ = url.PathUnescape(pathParts[len(pathParts)-1])
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
		itemPattern := regexp.MustCompile(`/mod_assign/[a-z_]+/(\d+)/`)
		if item := itemPattern.FindStringSubmatch(parsed.Path); len(item) > 1 {
			itemID = item[1]
		}
		result.Attachments = append(result.Attachments, Attachment{ID: id, Filename: filename, URL: rawURL, ItemID: itemID})
	}
	return result, nil
}

func textContent(raw []byte) string {
	document, err := html.Parse(strings.NewReader(string(raw)))
	if err != nil {
		return ""
	}
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
			text.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return strings.Join(strings.Fields(stdhtml.UnescapeString(text.String())), " ")
}

func AssignmentID(cmid string) string { return "K" + cmid }
func MaterialID(cmid string) string   { return "M" + cmid }

func MapModule(course model.Course, module Module) (assignment *model.Assignment, material *model.Material) {
	switch module.Type {
	case "assign", "quiz":
		return &model.Assignment{ID: AssignmentID(module.CMID), CourseID: course.ID, CourseCode: course.Short, Name: module.Name}, nil
	case "resource", "folder", "url", "book":
		kind := "file"
		if module.Type == "url" {
			kind = "url"
		}
		return nil, &model.Material{ID: MaterialID(module.CMID), CourseID: course.ID, CourseCode: course.Short, Name: module.Name, Section: module.Section, Kind: kind}
	default:
		return nil, nil
	}
}
