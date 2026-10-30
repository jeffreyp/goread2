// Package assets builds versioned URLs for static files so browsers can cache
// them for a long time and still fetch a new copy as soon as the content changes.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// hashLength is the number of hex characters of the SHA-256 digest used as
// the version, which is enough to make collisions between builds negligible.
const hashLength = 12

// Versioner maps a file below dir to a URL below prefix carrying a ?v= query
// parameter derived from the file's content. Hashes are computed on first use
// and cached for the life of the process, so a rebuilt file is picked up after
// a restart.
type Versioner struct {
	dir    string
	prefix string
	hashes sync.Map // name -> version string
}

// NewVersioner returns a Versioner for files in dir served at prefix,
// for example NewVersioner("web/static", "/static").
func NewVersioner(dir, prefix string) *Versioner {
	return &Versioner{dir: dir, prefix: prefix}
}

// URL returns the URL for name, a slash-separated path relative to the
// Versioner's directory such as "js/app.min.js". If the file cannot be read,
// URL logs the error and returns the URL without a version, so a missing
// build artifact degrades to an unversioned URL rather than a broken page.
// Failures are not cached, so the file is read again on the next call.
func (v *Versioner) URL(name string) string {
	url := v.prefix + "/" + name
	if version, ok := v.hashes.Load(name); ok {
		return url + "?v=" + version.(string)
	}

	data, err := os.ReadFile(filepath.Join(v.dir, filepath.FromSlash(name)))
	if err != nil {
		log.Printf("assets: cannot version %s: %v", name, err)
		return url
	}
	sum := sha256.Sum256(data)
	version := hex.EncodeToString(sum[:])[:hashLength]
	v.hashes.Store(name, version)
	return url + "?v=" + version
}
