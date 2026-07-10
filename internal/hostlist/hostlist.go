// Package hostlist loads domain lists and matches host names (SNI values)
// against them. A list entry matches the exact host and any of its subdomains,
// e.g. "youtube.com" matches "youtube.com" and "www.youtube.com" but not
// "notyoutube.com".
package hostlist

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// List is a set of domains to target.
type List struct {
	domains map[string]struct{}
}

// New returns an empty List.
func New() *List {
	return &List{domains: make(map[string]struct{})}
}

// Add inserts a single domain (case-insensitive, trimmed). Blank input is
// ignored.
func (l *List) Add(domain string) {
	d := normalize(domain)
	if d == "" {
		return
	}
	l.domains[d] = struct{}{}
}

// Len reports how many domains are loaded.
func (l *List) Len() int { return len(l.domains) }

// LoadReader adds every domain from r. Lines that are blank or begin with '#'
// are skipped. It returns the number of domains added.
func (l *List) LoadReader(r io.Reader) (int, error) {
	before := l.Len()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		l.Add(line)
	}
	if err := sc.Err(); err != nil {
		return l.Len() - before, err
	}
	return l.Len() - before, nil
}

// LoadFile adds every domain from the file at path.
func (l *List) LoadFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return l.LoadReader(f)
}

// Match reports whether host equals, or is a subdomain of, any list entry.
func (l *List) Match(host string) bool {
	h := normalize(host)
	if h == "" {
		return false
	}
	// Check the full host and each parent domain: for "a.b.youtube.com" test
	// "a.b.youtube.com", "b.youtube.com", "youtube.com", "com".
	for {
		if _, ok := l.domains[h]; ok {
			return true
		}
		i := strings.IndexByte(h, '.')
		if i < 0 {
			return false
		}
		h = h[i+1:]
	}
}

func normalize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(s, ".") // drop a trailing FQDN dot
	return s
}
