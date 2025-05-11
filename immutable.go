package pbimmutable

import (
	"fmt"
	"reflect"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// MakeImmutable returns a hook function that prevents changes to specified fields of a record.
// If no fieldNames are provided, all user-defined fields are considered immutable.
// System fields like "id", "created", and "updated" are always allowed to change (as PocketBase manages them).
func MakeImmutable(immutableFieldNames ...string) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Record == nil {
			return apis.NewBadRequestError("Record data is missing in the event.", nil)
		}
		if e.App == nil {
			return apis.NewBadRequestError("App context is missing in the event.", nil)
		}

		originalRecord, err := e.App.Dao().FindRecordById(e.Record.Collection().Id, e.Record.Id)
		if err != nil {
			return apis.NewBadRequestError(fmt.Sprintf("Failed to fetch original record %s from collection %s.", e.Record.Id, e.Record.Collection().Name), err)
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

			// DeepEqual is used for robust comparison, especially for slices, maps, etc.
			if !reflect.DeepEqual(originalValue, pendingValue) {
				// Check if the field is a system field that is allowed to change (e.g. "updated")
				// This check is more of a safeguard, primary filtering is done above for the "all fields" case.
				if isSystemField(fieldName) && fieldName == models.SystemFieldUpdated {
					continue // Allow 'updated' field to change
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
		return nil
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
