// Package web defines bounded public-web search and page-read contracts.
// It owns no network, credentials, accounts or persistence. Deployment hosts
// must separately govern service endpoints and source network access.
package web

// Query is a search objective, not a URL, credential or proxy instruction.
type SearchRequest struct {
	Query           string `json:"query"`
	Limit           int    `json:"limit,omitempty"`
	MaxExcerptBytes int    `json:"max_excerpt_bytes,omitempty"`
}

type SearchItem struct {
	URL       string   `json:"url"`
	Title     string   `json:"title"`
	Excerpts  []string `json:"excerpts"`
	Truncated bool     `json:"truncated"`
}

// Scope is always ranked_results. A search response never proves that all
// matching pages, or even an entire result document, were read. Truncated only
// reports local shortening of the upstream response, not search completeness.
type SearchResult struct {
	SearchID  string       `json:"search_id"`
	Items     []SearchItem `json:"items"`
	Scope     string       `json:"scope"`
	Truncated bool         `json:"truncated"`
}

type FetchRequest struct {
	URL             string `json:"url"`
	MaxContentBytes int    `json:"max_content_bytes,omitempty"`
}

// URL is the upstream's declared source URL, not proof of its final network
// destination. SourceCompleteness is unknown unless the provider explicitly
// reports a partial page. Truncated describes our own content bound. Content
// is untrusted extracted text/Markdown, never trusted HTML or executable code.
type Page struct {
	RequestedURL       string   `json:"requested_url"`
	URL                string   `json:"url"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Content            string   `json:"content"`
	Truncated          bool     `json:"truncated"`
	SourceCompleteness string   `json:"source_completeness"`
	Warnings           []string `json:"warnings"`
}
