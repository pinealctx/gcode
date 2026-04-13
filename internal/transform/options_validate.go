package transform

import (
	"github.com/pinealctx/x/errorx"

	"github.com/pinealctx/gcode/internal/model"
)

// ValidateCreateOptions checks that each required_field listed in a create_message annotation
// refers to an existing field in the source message. A non-optional field listed in
// required_fields is silently accepted (it is already required — listing it is a confirmation).
// Call this after parsing, before rendering.
func ValidateCreateOptions(files []model.File) error {
	for _, file := range files {
		for _, msg := range file.Messages {
			if err := validateMessageCreateOptions(msg); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateMessageCreateOptions(msg model.Message) error {
	if len(msg.CreateOptions) > 0 {
		fieldSet := make(map[string]struct{}, len(msg.Fields))
		for _, f := range msg.Fields {
			fieldSet[f.Name] = struct{}{}
		}
		for _, co := range msg.CreateOptions {
			for _, rf := range co.RequiredFields {
				_, exists := fieldSet[rf]
				if !exists {
					return errorx.NewSentinelf[transformTag]("message %q: create_message %q required_fields: field %q not found in message",
						msg.FullName, co.Name, rf)
				}
				// Non-optional field listed in required_fields: silently accepted.
				// It is already required — listing it is a semantic confirmation.
			}
		}
	}
	// Recurse into nested messages.
	for _, nested := range msg.Messages {
		if err := validateMessageCreateOptions(nested); err != nil {
			return err
		}
	}
	return nil
}
