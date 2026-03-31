package validateruntime

import (
	"net/mail"
	"testing"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"

	pbgo "github.com/pinealctx/gcode/testdata/compat/pbgo"
)

// stdlibIsEmail uses net/mail.ParseAddress (RFC 5322, accepts "Name <email>" format).
func stdlibIsEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}

var benchEmails = []struct {
	name  string
	email string
}{
	{"valid", "user@example.com"},
	{"invalid", "not-an-email"},
}

func BenchmarkIsEmail_Default(b *testing.B) {
	for _, tc := range benchEmails {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				defaultIsEmail(tc.email)
			}
		})
	}
}

func BenchmarkIsEmail_Stdlib(b *testing.B) {
	for _, tc := range benchEmails {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				stdlibIsEmail(tc.email)
			}
		})
	}
}

func BenchmarkIsEmail_Protovalidate(b *testing.B) {
	v, err := protovalidate.New()
	if err != nil {
		b.Fatal(err)
	}
	for _, tc := range benchEmails {
		b.Run(tc.name, func(b *testing.B) {
			msg := &pbgo.Person{Email: tc.email}
			for b.Loop() {
				_ = v.Validate(proto.Message(msg))
			}
		})
	}
}
