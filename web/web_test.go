package web_test

import (
	"strings"
	"testing"

	web "github.com/domainry/domainry-connector-sdk/web"
)

func TestContractIdentity(t *testing.T) {
	if got := web.ComputedContractSHA256(); got != web.ContractSHA256 {
		t.Fatalf("web contract identity %s", got)
	}
	if web.OperationSHA256("unknown") != "" {
		t.Fatal("unknown operation has identity")
	}
	for key, want := range map[string]string{web.SearchOperationKey: "8c24df34c2968ac2fac33fe5091f5f0d606ea26b9315def2cddd3505fa4063d0", web.FetchOperationKey: "59bc1d78623a0ab72320efe81bd601592149719b8c0957509b07cb0ddbee8b06"} {
		if got := web.OperationSHA256(key); got != want {
			t.Fatalf("operation %s hash %s", key, got)
		}
	}
}
func TestPublicURLSyntaxAndNormalization(t *testing.T) {
	for in, want := range map[string]string{
		"https://Example.COM":                 "https://example.com/",
		"https://example.com:443/a?b=2&a=1":   "https://example.com/a?b=2&a=1",
		"http://example.com:80/a#section":     "http://example.com/a",
		"https://example.com/中文?q=%E4%B8%AD":  "https://example.com/%E4%B8%AD%E6%96%87?q=%E4%B8%AD",
		"https://xn--fiqs8s.example.com/page": "https://xn--fiqs8s.example.com/page",
		"https://8.8.8.8":                     "https://8.8.8.8/",
		"https://[2001:4860:4860::8888]/":     "https://[2001:4860:4860::8888]/",
	} {
		got, err := web.NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("normalize %q: %q %v", in, got, err)
		}
	}
	for _, in := range []string{"", " https://example.com", "file:///etc/passwd", "ftp://example.com/a", "javascript:alert(1)", "//example.com/a", "https://user:pass@example.com", "https://example.com:8080", "https://example.com:", "http://localhost", "http://foo.localhost", "http://foo.local", "http://foo.internal", "http://foo.home.arpa", "http://foo.test", "http://foo.onion", "http://foo", "http://127.0.0.1", "http://10.1.1.1", "http://169.254.169.254/latest", "http://100.64.0.1", "http://192.0.2.1", "http://198.51.100.1", "http://203.0.113.1", "http://198.18.0.1", "http://0.0.0.0", "http://224.0.0.1", "http://255.255.255.255", "http://[::1]", "http://[::ffff:127.0.0.1]", "http://[fd00::1]", "http://[fe80::1%25en0]", "http://[2001:db8::1]", "http://[64:ff9b::7f00:1]", "http://[2002:7f00:1::1]", "http://127.1", "http://2130706433", "http://0x7f000001", "http://0x7f.0.0.0x1", "http://0177.0.0.1", "https://example.com./", "https://-example.com", "https://a..com", "https://例子.com", "https://example.com/%00", "https://example.com/?q=%0A", "https://example.com/%ff", "https://example.com/\\foo", "https://example.com/?", "https://example.com/ space", "https://example.com/#%00", "https://example.com/" + strings.Repeat("中", 500)} {
		if got, err := web.NormalizeURL(in); err == nil {
			t.Errorf("unsafe URL accepted %q => %q", in, got)
		}
	}
	// A syntactically public name is not a DNS or network authorization decision.
	if _, err := web.NormalizeURL("https://unresolved.example.com/path"); err != nil {
		t.Fatal(err)
	}
}
func TestRequestLimits(t *testing.T) {
	request := web.SearchRequest{Query: "今天的版本发布"}
	if request.Validate() != nil || request.ResultLimit() != 5 || request.ExcerptLimit() != 1500 {
		t.Fatal("search defaults")
	}
	for _, r := range []web.SearchRequest{{}, {Query: " \t"}, {Query: "a\n"}, {Query: strings.Repeat("中", 1366)}, {Query: "q", Limit: -1}, {Query: "q", Limit: 11}, {Query: "q", MaxExcerptBytes: 99}, {Query: "q", MaxExcerptBytes: 8001}} {
		if r.Validate() == nil {
			t.Fatalf("bad search accepted %+v", r)
		}
	}
	fetch := web.FetchRequest{URL: "https://example.com"}
	if fetch.Validate() != nil || fetch.ContentLimit() != 16384 {
		t.Fatal("fetch defaults")
	}
	for _, limit := range []int{-1, 1, 1023, 65537} {
		if (web.FetchRequest{URL: fetch.URL, MaxContentBytes: limit}).Validate() == nil {
			t.Fatalf("bad content limit %d", limit)
		}
	}
}
func TestSearchEvidenceAndBounds(t *testing.T) {
	request := web.SearchRequest{Query: "current", Limit: 1, MaxExcerptBytes: 100}
	base := func() web.SearchResult {
		return web.SearchResult{SearchID: "s-1", Scope: "ranked_results", Items: []web.SearchItem{{URL: "https://example.com/", Title: "来源", Excerpts: []string{"片段"}}}}
	}
	if base().Validate(request) != nil {
		t.Fatal("valid evidence")
	}
	empty := base()
	empty.Items = []web.SearchItem{}
	if empty.Validate(request) != nil {
		t.Fatal("empty search is valid")
	}
	for name, change := range map[string]func(*web.SearchResult){
		"missing ID":       func(r *web.SearchResult) { r.SearchID = "" },
		"complete claim":   func(r *web.SearchResult) { r.Scope = "all_results" },
		"missing items":    func(r *web.SearchResult) { r.Items = nil },
		"over result cap":  func(r *web.SearchResult) { r.Items = append(r.Items, r.Items[0]) },
		"private URL":      func(r *web.SearchResult) { r.Items[0].URL = "http://127.0.0.1/" },
		"unnormalized URL": func(r *web.SearchResult) { r.Items[0].URL = "https://EXAMPLE.com/" },
		"oversize title":   func(r *web.SearchResult) { r.Items[0].Title = strings.Repeat("中", 342) },
		"missing excerpts": func(r *web.SearchResult) { r.Items[0].Excerpts = nil },
		"aggregate bytes": func(r *web.SearchResult) {
			r.Items[0].Excerpts = []string{strings.Repeat("a", 60), strings.Repeat("a", 60)}
		},
		"invalid UTF8":      func(r *web.SearchResult) { r.Items[0].Excerpts = []string{string([]byte{255})} },
		"truncation hidden": func(r *web.SearchResult) { r.Items[0].Truncated = true },
	} {
		r := base()
		change(&r)
		if r.Validate(request) == nil {
			t.Error(name)
		}
	}
	r := base()
	r.Items = append(r.Items, r.Items[0])
	request.Limit = 2
	if r.Validate(request) == nil {
		t.Fatal("duplicate URL accepted")
	}
	r = base()
	r.Truncated = true
	r.Items[0].Truncated = true
	if r.Validate(request) != nil {
		t.Fatal("explicit truncation rejected")
	}
}
func TestPageSourceAndCompleteness(t *testing.T) {
	request := web.FetchRequest{URL: "https://example.com/page#heading", MaxContentBytes: 1024}
	base := func() web.Page {
		return web.Page{RequestedURL: "https://example.com/page", URL: "https://example.com/canonical", Title: "Source", Content: "Untrusted\nMarkdown\ttext", Warnings: []string{}, SourceCompleteness: "unknown"}
	}
	if base().Validate(request) != nil {
		t.Fatal("valid source page")
	}
	for name, change := range map[string]func(*web.Page){
		"wrong request":              func(p *web.Page) { p.RequestedURL = "https://other.com/" },
		"private source":             func(p *web.Page) { p.URL = "http://10.0.0.1/" },
		"unsupported complete claim": func(p *web.Page) { p.SourceCompleteness = "complete" },
		"missing warnings":           func(p *web.Page) { p.Warnings = nil },
		"invalid warning":            func(p *web.Page) { p.Warnings = []string{"warning\nsecret"} },
		"oversized content":          func(p *web.Page) { p.Content = strings.Repeat("中", 342); p.Truncated = true },
		"control content":            func(p *web.Page) { p.Content = "bad\x00" },
	} {
		p := base()
		change(&p)
		if p.Validate(request) == nil {
			t.Error(name)
		}
	}
	p := base()
	p.Content = ""
	p.SourceCompleteness = "partial"
	p.Warnings = []string{"upstream warning"}
	if p.Validate(request) != nil {
		t.Fatal("partial empty body rejected")
	}
}
