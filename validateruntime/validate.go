// Package validateruntime provides runtime helpers for generated Validate() methods.
// It includes ValidationError, email/URI validators, and a pattern-matching cache.
package validateruntime

import (
	"net/url"
	"regexp"
	"sync"
)

// ValidationError is returned by generated Validate() methods when a field
// fails a constraint. Field contains the full dot-separated path to the field
// (e.g. "address.city"), Rule names the constraint (e.g. "email", "min_len"),
// and Message is a human-readable description.
type ValidationError struct {
	Field   string
	Rule    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// defaultEmailRegexp is the pre-compiled regex for defaultIsEmail.
// Local part: word chars, supports - + . separators.
// Domain: valid label format, TLD 2-63 letters.
var defaultEmailRegexp = regexp.MustCompile(
	`^[\w]+([-+.]\w+)*@([A-Za-z0-9]([-_A-Za-z0-9]*[A-Za-z0-9])?\.)+[A-Za-z]{2,63}$`,
)

func defaultIsEmail(s string) bool {
	return defaultEmailRegexp.MatchString(s)
}

func defaultIsURI(s string) bool {
	_, err := url.ParseRequestURI(s)
	return err == nil
}

// EmailValidator is the function used by IsEmail. Replace it in init() to
// swap the email validation implementation.
var EmailValidator = defaultIsEmail

// URIValidator is the function used by IsURI. Replace it in init() to
// swap the URI validation implementation.
var URIValidator = defaultIsURI

// IsEmail reports whether s is a valid email address.
// Delegates to EmailValidator, which defaults to a pre-compiled regex.
func IsEmail(s string) bool { return EmailValidator(s) }

// IsURI reports whether s is a valid URI (RFC 3986).
// Delegates to URIValidator, which defaults to net/url.ParseRequestURI.
func IsURI(s string) bool { return URIValidator(s) }

// patternCache caches compiled *regexp.Regexp by pattern string.
var patternCache sync.Map

// MsgOr returns override if non-empty, otherwise defaultMsg.
// Used by generated Validate() methods to apply per-field message overrides.
func MsgOr(override, defaultMsg string) string {
	if override != "" {
		return override
	}
	return defaultMsg
}

// Compiled regexps are cached in a package-level sync.Map; the pattern is
// guaranteed to be valid because the parser validates it at code-generation time.
func MatchPattern(s, pattern string) bool {
	var re *regexp.Regexp
	if v, ok := patternCache.Load(pattern); ok {
		re, ok = v.(*regexp.Regexp)
		if !ok || re == nil {
			return false
		}
	} else {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		actual, _ := patternCache.LoadOrStore(pattern, compiled)
		re, ok = actual.(*regexp.Regexp)
		if !ok || re == nil {
			return false
		}
	}
	return re.MatchString(s)
}
