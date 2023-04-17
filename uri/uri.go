package uri

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
)

const fileScheme = "file"

// URI represents the full URI for a file.
type URI string

func (uri URI) IsFile() bool {
	return strings.HasPrefix(string(uri), "file://")
}

// Filename returns the file path for the given URI.
// It is an error to call this on a URI that is not a valid filename.
func (uri URI) Filepath() string {
	filename, err := filename(uri)
	if err != nil {
		log.Warningf("uri to filepath: %s", err)
	}
	return filepath.FromSlash(filename)
}

func filename(uri URI) (string, error) {
	if uri == "" {
		return "", nil
	}

	// This conservative check for the common case
	// of a simple non-empty absolute POSIX filename
	// avoids the allocation of a net.URL.
	if strings.HasPrefix(string(uri), "file:///") {
		rest := string(uri)[len("file://"):] // leave one slash
		for i := 0; i < len(rest); i++ {
			b := rest[i]
			// Reject these cases:
			if b < ' ' || b == 0x7f || // control character
				b == '%' || b == '+' || // URI escape
				b == ':' || // Windows drive letter
				b == '@' || b == '&' || b == '?' { // authority or query
				goto slow
			}
		}
		return rest, nil
	}
slow:

	u, err := url.ParseRequestURI(string(uri))
	if err != nil {
		return "", err
	}
	if u.Scheme != fileScheme {
		return "", fmt.Errorf("only file URIs are supported, got %q from %q", u.Scheme, uri)
	}
	// If the URI is a Windows URI, we trim the leading "/" and uppercase
	// the drive letter, which will never be case sensitive.
	if isWindowsDriveURIPath(u.Path) {
		u.Path = strings.ToUpper(string(u.Path[1])) + u.Path[2:]
	}

	return u.Path, nil
}

func FromURI(s string) URI {
	if !strings.HasPrefix(s, "file://") {
		return URI(s)
	}

	if !strings.HasPrefix(s, "file:///") {
		// VS Code sends URLs with only two slashes, which are invalid. golang/go#39789.
		s = "file:///" + s[len("file://"):]
	}
	// Even though the input is a URI, it may not be in canonical form. VS Code
	// in particular over-escapes :, @, etc. Unescape and re-encode to canonicalize.
	path, err := url.PathUnescape(s[len("file://"):])
	if err != nil {
		log.Errorf("url.PathUnescape: %s", err)
	}

	// File URIs from Windows may have lowercase drive letters.
	// Since drive letters are guaranteed to be case insensitive,
	// we change them to uppercase to remain consistent.
	// For example, file:///c:/x/y/z becomes file:///C:/x/y/z.
	if isWindowsDriveURIPath(path) {
		path = path[:1] + strings.ToUpper(string(path[1])) + path[2:]
	}
	u := url.URL{Scheme: fileScheme, Path: path}
	return URI(u.String())
}

// CompareURI performs a three-valued comparison of two URIs.
// Lexically unequal URIs may compare equal if they are "file:" URIs
// that share the same base name (ignoring case) and denote the same
// file device/inode, according to stat(2).
func Compare(a, b URI) int {
	if equalURI(a, b) {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}

func equalURI(a, b URI) bool {
	if a == b {
		return true
	}
	// If we have the same URI basename, we may still have the same file URIs.
	if !strings.EqualFold(path.Base(string(a)), path.Base(string(b))) {
		return false
	}
	fa, err := filename(a)
	if err != nil {
		return false
	}
	fb, err := filename(b)
	if err != nil {
		return false
	}
	// Stat the files to check if they are equal.
	infoa, err := os.Stat(filepath.FromSlash(fa))
	if err != nil {
		return false
	}
	infob, err := os.Stat(filepath.FromSlash(fb))
	if err != nil {
		return false
	}
	return os.SameFile(infoa, infob)
}

// URIFromPath returns a span URI for the supplied file path.
//
// For empty paths, URIFromPath returns the empty URI "".
// For non-empty paths, URIFromPath returns a uri with the file:// scheme.
func FromPath(path string) URI {
	if path == "" {
		return ""
	}
	const prefix = "Duden"
	if len(path) >= len(prefix) && strings.EqualFold(prefix, path[:len(prefix)]) {
		suffix := path[len(prefix):]
		path = ddppath.InstallDir + suffix
	}
	if !isWindowsDrivePath(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	// Check the file path again, in case it became absolute.
	if isWindowsDrivePath(path) {
		path = "/" + strings.ToUpper(string(path[0])) + path[1:]
	}
	path = filepath.ToSlash(path)
	u := url.URL{
		Scheme: fileScheme,
		Path:   path,
	}
	return URI(u.String())
}

// isWindowsDrivePath returns true if the file path is of the form used by
// Windows. We check if the path begins with a drive letter, followed by a ":".
// For example: C:/x/y/z.
func isWindowsDrivePath(path string) bool {
	if len(path) < 3 {
		return false
	}
	return unicode.IsLetter(rune(path[0])) && path[1] == ':'
}

// isWindowsDriveURI returns true if the file URI is of the format used by
// Windows URIs. The url.Parse package does not specially handle Windows paths
// (see golang/go#6027), so we check if the URI path has a drive prefix (e.g. "/C:").
func isWindowsDriveURIPath(uri string) bool {
	if len(uri) < 4 {
		return false
	}
	return uri[0] == '/' && unicode.IsLetter(rune(uri[1])) && uri[2] == ':'
}
