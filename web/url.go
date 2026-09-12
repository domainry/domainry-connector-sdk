package web

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
	"unicode"
)

// NormalizeURL validates public-web URL syntax only. It performs no DNS lookup
// and cannot authorize a hostname, follow redirects or prevent DNS rebinding.
// Those policies belong to the deployment and remote fetch service. DNS names
// must be ASCII (IDNs use punycode), standard ports only, no embedded credentials.
func NormalizeURL(raw string) (string, error) {
	invalid := errors.New("invalid public web URL")
	if raw == "" || strings.TrimSpace(raw) != raw || !text(raw, 4096, false) || strings.ContainsFunc(raw, unicode.IsSpace) || strings.Contains(raw, "\\") {
		return "", invalid
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.Host == "" || u.ForceQuery {
		return "", invalid
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(u.Host, ":") || strings.HasSuffix(host, ".") || strings.Contains(host, "%") || u.Port() != "" && !(u.Scheme == "https" && u.Port() == "443" || u.Scheme == "http" && u.Port() == "80") {
		return "", invalid
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return "", invalid
		}
		for _, cidr := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "2001::/23", "2002::/16", "64:ff9b::/96", "64:ff9b:1::/48"} {
			if netip.MustParsePrefix(cidr).Contains(ip) {
				return "", invalid
			}
		}
		host = ip.String()
		if ip.Is6() {
			host = "[" + host + "]"
		}
	} else {
		if !validDNSName(host) {
			return "", invalid
		}
	}
	// Default ports add no identity information; remove them consistently.
	if !text(u.Fragment, 4096, false) {
		return "", invalid
	}
	u.Fragment, u.RawFragment = "", ""
	u.Host = host
	if u.Path == "" {
		u.Path = "/"
	}
	// Reject escaped controls as well, while preserving the remaining exact query.
	decoded, err := url.PathUnescape(u.EscapedPath())
	if err != nil || !text(decoded, 4096, false) || strings.Contains(decoded, "\\") {
		return "", invalid
	}
	query, err := url.QueryUnescape(u.RawQuery)
	if err != nil || !text(query, 4096, false) {
		return "", invalid
	}
	normalized := u.String()
	if len(normalized) > 4096 {
		return "", invalid
	}
	return normalized, nil
}
func validDNSName(host string) bool {
	if len(host) > 253 || !strings.Contains(host, ".") {
		return false
	}
	for _, suffix := range []string{".localhost", ".local", ".internal", ".test", ".invalid", ".example", ".onion", ".home.arpa", ".lan", ".corp", ".intranet"} {
		if strings.HasSuffix(host, suffix) {
			return false
		}
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	// A numeric final label could be an alternative IP representation. Decimal,
	// octal and abbreviated IPv4 forms must not be interpreted as DNS hostnames.
	last := labels[len(labels)-1]
	return last[0] >= 'a' && last[0] <= 'z'
}
