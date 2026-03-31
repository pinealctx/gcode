package transform

import (
	"fmt"

	"github.com/pinealctx/gcode/internal/model"
)

// ValidateCreateOptions checks that each required_field listed in a create_message annotation
// refers to an existing field that is optional in the source message.
// A non-optional field is already required, so listing it in required_fields is an error.
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
		optionalSet := make(map[string]bool, len(msg.Fields))
		for _, f := range msg.Fields {
			optionalSet[f.Name] = f.Optional
		}
		for _, co := range msg.CreateOptions {
			for _, rf := range co.RequiredFields {
				isOpt, exists := optionalSet[rf]
				if !exists {
					return fmt.Errorf("message %q: create_message %q required_fields: field %q not found in message",
						msg.FullName, co.Name, rf)
				}
				if !isOpt {
					return fmt.Errorf("message %q: create_message %q required_fields: field %q is already non-optional",
						msg.FullName, co.Name, rf)
				}
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
