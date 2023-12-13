package main

// HTTPResult URL，status_code，title, redirect_url(如果有302、301等状态)
type HTTPResult struct {
	URL         string
	StatusCode  int
	Title       string
	RedirectURL string
}
