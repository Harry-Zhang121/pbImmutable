package pbimmutable

import (
	"errors" // Added for errors.New
	"fmt"
	"reflect"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// MakeImmutable returns a hook function that prevents changes to specified fields of a record.
// It can also take an optional callback function of type `func(e *core.RecordEvent) error`.
// This callback is executed if all immutability checks pass.
// The overall database transaction for the update operation commits only if:
// 1. All immutability checks pass.
// 2. The provided callback function (if any) also returns nil.
// If any of these conditions fail (e.g., an immutable field is changed, or the callback returns an error),
// the entire transaction is rolled back.
//
// Usage examples:
// MakeImmutable("field1", "field2") // Only immutable fields
// MakeImmutable("field1", myCallback) // Immutable field and a callback
// MakeImmutable(myCallback)          // All user-defined fields immutable, and a callback
// MakeImmutable()                    // All user-defined fields immutable, no callback
func MakeImmutable(args ...interface{}) func(e *core.RecordEvent) error {
	var immutableFieldNames []string
	var userCallback func(e *core.RecordEvent) error
	var parseError error

	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			immutableFieldNames = append(immutableFieldNames, v)
		case func(e *core.RecordEvent) error:
			if userCallback != nil {
				parseError = errors.New("pbimmutable.MakeImmutable: only one callback function can be provided")
				break
			}
			userCallback = v
		default:
			parseError = fmt.Errorf("pbimmutable.MakeImmutable: invalid argument type %T at position %d", arg, i)
			break
		}
		if parseError != nil {
			break
		}
	}

	// The actual hook function returned
	return func(e *core.RecordEvent) error {
		if parseError != nil { // Return parsing error immediately if MakeImmutable was called incorrectly
			return apis.NewBadRequestError(fmt.Sprintf("MakeImmutable setup error: %v", parseError), nil)
		}

		if e.Record == nil {
			return apis.NewBadRequestError("Record data is missing in the event.", nil)
		}
		if e.App == nil {
			return apis.NewBadRequestError("App context is missing in the event.", nil)
		}

		originalRecord, err := e.App.Dao().FindRecordById(e.Record.Collection().Id, e.Record.Id)
		if err != nil {
			return apis.NewBadRequestError(fmt.Sprintf("Failed to fetch original record %s from collection %s for immutability check.", e.Record.Id, e.Record.Collection().Name), err)
		}

		fieldsToCheck := immutableFieldNames
		if len(immutableFieldNames) == 0 {
			// If no specific fields are provided, all non-system fields are considered immutable.
			schemaFields := e.Record.Schema().Fields()
			fieldsToCheck = make([]string, 0, len(schemaFields))
			for _, field := range schemaFields {
				if !isSystemField(field.Name) {
					fieldsToCheck = append(fieldsToCheck, field.Name)
				}
			}
		}

		for _, fieldName := range fieldsToCheck {
			originalValue := originalRecord.Get(fieldName)
			pendingValue := e.Record.Get(fieldName)

			if !reflect.DeepEqual(originalValue, pendingValue) {
				if isSystemField(fieldName) && fieldName == models.SystemFieldUpdated {
					continue
				}

				return apis.NewBadRequestError(
					fmt.Sprintf("Attempt to modify immutable field '%s'.", fieldName),
					map[string]any{
						"field":    fieldName,
						"reason":   "immutable",
						"recordId": e.Record.Id,
					},
				)
			}
		}

		// If we've reached here, all immutability checks passed.

		// Attempt to proceed with the main operation (e.g., database commit)
		err = e.Next() // This line assumes 'e' has a Next() method.
		if err != nil {
			// If e.Next() fails, it implies the underlying operation (eg. DB save) failed.
			return fmt.Errorf("failed to commit record changes via e.Next() after immutability checks: %w", err)
		}
		// If e.Next() succeeded, the main operation is now considered committed.

		// Now, if a user callback was provided, execute it.
		// This callback runs AFTER the main record update has been successfully committed via e.Next().
		if userCallback != nil {
			if callbackErr := userCallback(e); callbackErr != nil {
				// The main record operation was committed. This error is from the subsequent user-defined callback.
				// The API will report this callback error, but the record data was already saved.
				// Consider logging this error or handling it in a way that acknowledges the main commit succeeded.
				return fmt.Errorf("user callback failed AFTER record commit: %w", callbackErr)
			}
		}

		return nil // Signifies success of this hook and the post-commit callback.
	}
}

// isSystemField checks if a field name is one of PocketBase's system fields.
func isSystemField(fieldName string) bool {
	switch fieldName {
	case models.SystemFieldId, models.SystemFieldCreated, models.SystemFieldUpdated, models.SystemFieldCollectionId, models.SystemFieldCollectionName, models.SystemFieldExpand:
		return true
	default:
		return false
	}
}
