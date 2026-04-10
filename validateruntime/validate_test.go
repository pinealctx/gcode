package validateruntime

import (
	"testing"
)

func TestValidationError(t *testing.T) {
	t.Parallel()
	e := &ValidationError{Field: "email", Rule: "email", Message: "must be a valid email address"}
	if got := e.Error(); got != "email: must be a valid email address" {
		t.Errorf("Error() = %q, want %q", got, "email: must be a valid email address")
	}
}

func TestIsEmail(t *testing.T) {
	t.Parallel()
	valid := []string{
		"user@example.com",
		"user+tag@example.co.uk",
		"user.name@sub.domain.org",
		"user-name@example.io",
	}
	for _, s := range valid {
		if !IsEmail(s) {
			t.Errorf("IsEmail(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",
		"notanemail",
		"@nodomain.com",
		"user@",
		"user @example.com",
		"user@.com",
	}
	for _, s := range invalid {
		if IsEmail(s) {
			t.Errorf("IsEmail(%q) = true, want false", s)
		}
	}
}

func TestIsURI(t *testing.T) {
	t.Parallel()
	valid := []string{
		// common HTTP(S)
		"https://example.com",
		"http://example.com/path?q=1#fragment",
		"http://user:pass@example.com:8080/path",
		// other standard schemes
		"ftp://ftp.example.com/pub/file.txt",
		"ftps://ftp.example.com/secure",
		"mailto:user@example.com",
		"urn:isbn:0451450523",
		"data:text/plain;base64,SGVsbG8=",
		"file:///etc/hosts",
		// relative paths (accepted by ParseRequestURI)
		"/relative/path",
		"/path/with?query=1",
	}
	for _, s := range valid {
		if !IsURI(s) {
			t.Errorf("IsURI(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",
		"not a uri",
		"://missing-scheme",
		"just-a-word",
		"example.com", // no scheme, no leading slash
	}
	for _, s := range invalid {
		if IsURI(s) {
			t.Errorf("IsURI(%q) = true, want false", s)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	t.Parallel()
	if !MatchPattern("abc123", `^[a-z]+\d+$`) {
		t.Error("MatchPattern(abc123, ^[a-z]+\\d+$) = false, want true")
	}
	if MatchPattern("ABC123", `^[a-z]+\d+$`) {
		t.Error("MatchPattern(ABC123, ^[a-z]+\\d+$) = true, want false")
	}
	// invalid pattern returns false, no panic
	if MatchPattern("abc", `[invalid`) {
		t.Error("MatchPattern with invalid pattern should return false")
	}
}

func TestMatchPatternCaching(t *testing.T) {
	t.Parallel()
	pattern := `^\d{4}-\d{2}-\d{2}$`
	// call twice to exercise cache path
	if !MatchPattern("2024-01-15", pattern) {
		t.Error("first call: want true")
	}
	if !MatchPattern("2024-12-31", pattern) {
		t.Error("second call (cached): want true")
	}
	if MatchPattern("not-a-date", pattern) {
		t.Error("non-matching: want false")
	}
}

// TestMsgOr verifies that MsgOr returns the override when non-empty and the
// default message when the override is empty.
func TestMsgOr(t *testing.T) {
	t.Parallel()
	// override non-empty: returns override.
	if got := MsgOr("custom message", "default"); got != "custom message" {
		t.Errorf("MsgOr(non-empty, default) = %q, want %q", got, "custom message")
	}
	// override empty: returns defaultMsg.
	if got := MsgOr("", "default"); got != "default" {
		t.Errorf("MsgOr(\"\", default) = %q, want %q", got, "default")
	}
}

// TestURIValidatorReplacement verifies that replacing URIValidator changes the
// behavior of IsURI, mirroring the pattern used by TestEmailValidatorReplacement.
// Not parallel: modifies package-level URIValidator variable.
func TestURIValidatorReplacement(t *testing.T) {
	orig := URIValidator
	defer func() { URIValidator = orig }()

	URIValidator = func(s string) bool { return s == "allowed://uri" }
	if !IsURI("allowed://uri") {
		t.Error("replaced URIValidator: want true for allowed://uri")
	}
	if IsURI("https://example.com") {
		t.Error("replaced URIValidator: want false for https://example.com")
	}
}

// TestEmailValidatorReplacement verifies that replacing EmailValidator changes
// the behavior of IsEmail.
// Not parallel: modifies package-level EmailValidator variable.
func TestEmailValidatorReplacement(t *testing.T) {
	orig := EmailValidator
	defer func() { EmailValidator = orig }()

	EmailValidator = func(s string) bool { return s == "allowed@test.com" }
	if !IsEmail("allowed@test.com") {
		t.Error("replaced validator: want true for allowed@test.com")
	}
	if IsEmail("user@example.com") {
		t.Error("replaced validator: want false for user@example.com")
	}
}
