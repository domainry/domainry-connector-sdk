package web

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultSearchLimit  = 5
	MaximumSearchLimit  = 10
	DefaultExcerptBytes = 1500
	MaximumExcerptBytes = 8000
	DefaultContentBytes = 16 << 10
	MaximumContentBytes = 64 << 10
)

func text(s string, maximum int, multiline bool) bool {
	return len(s) <= maximum && utf8.ValidString(s) && !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\t')) })
}
func (r SearchRequest) ResultLimit() int {
	if r.Limit == 0 {
		return DefaultSearchLimit
	}
	return r.Limit
}
func (r SearchRequest) ExcerptLimit() int {
	if r.MaxExcerptBytes == 0 {
		return DefaultExcerptBytes
	}
	return r.MaxExcerptBytes
}
func (r SearchRequest) Validate() error {
	if strings.TrimSpace(r.Query) == "" || !text(r.Query, 4096, false) || r.Limit < 0 || r.Limit > MaximumSearchLimit || r.MaxExcerptBytes != 0 && (r.MaxExcerptBytes < 100 || r.MaxExcerptBytes > MaximumExcerptBytes) {
		return errors.New("invalid web search request")
	}
	return nil
}
func (r FetchRequest) ContentLimit() int {
	if r.MaxContentBytes == 0 {
		return DefaultContentBytes
	}
	return r.MaxContentBytes
}
func (r FetchRequest) Validate() error {
	if _, err := NormalizeURL(r.URL); err != nil {
		return err
	}
	if r.MaxContentBytes != 0 && (r.MaxContentBytes < 1024 || r.MaxContentBytes > MaximumContentBytes) {
		return errors.New("invalid web content limit")
	}
	return nil
}
func (r SearchResult) Validate(request SearchRequest) error {
	if request.Validate() != nil || r.SearchID == "" || strings.TrimSpace(r.SearchID) != r.SearchID || !text(r.SearchID, 1024, false) || r.Scope != "ranked_results" || r.Items == nil || len(r.Items) > request.ResultLimit() {
		return errors.New("invalid web search result")
	}
	seen := map[string]bool{}
	for _, item := range r.Items {
		normalized, err := NormalizeURL(item.URL)
		if err != nil || normalized != item.URL || seen[item.URL] || !text(item.Title, 1024, false) || item.Excerpts == nil || len(item.Excerpts) > 16 {
			return errors.New("invalid web search item")
		}
		seen[item.URL] = true
		bytes := 0
		for _, excerpt := range item.Excerpts {
			if !text(excerpt, request.ExcerptLimit(), true) {
				return errors.New("invalid web excerpt")
			}
			bytes += len(excerpt)
		}
		if bytes > request.ExcerptLimit() {
			return errors.New("web excerpt limit exceeded")
		}
		if item.Truncated && !r.Truncated {
			return errors.New("missing web truncation flag")
		}
	}
	return nil
}
func (p Page) Validate(request FetchRequest) error {
	if request.Validate() != nil {
		return errors.New("invalid web page request")
	}
	requested, _ := NormalizeURL(request.URL)
	source, err := NormalizeURL(p.URL)
	if err != nil || source != p.URL || requested != p.RequestedURL || !text(p.Title, 1024, false) || !text(p.Description, 4096, false) || !text(p.Content, request.ContentLimit(), true) || p.Warnings == nil || len(p.Warnings) > 8 {
		return errors.New("invalid web page")
	}
	if p.SourceCompleteness != "unknown" && p.SourceCompleteness != "partial" {
		return errors.New("invalid web source completeness")
	}
	for _, warning := range p.Warnings {
		if strings.TrimSpace(warning) == "" || !text(warning, 512, false) {
			return errors.New("invalid web warning")
		}
	}
	return nil
}
