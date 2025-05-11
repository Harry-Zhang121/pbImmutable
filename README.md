# PocketBase Immutable Fields Plugin

[![Go Reference](https://pkg.go.dev/badge/github.com/Harry-Zhang121/pbimmutable.svg)](https://pkg.go.dev/github.com/Harry-Zhang121/pbimmutable) 

This library provides a flexible way to make specific record fields immutable in your [PocketBase](https://pocketbase.io/) application and to run custom logic after immutability checks pass but before the record update is committed.

## Installation

To use this library in your PocketBase project, you can `go get` it:

```bash
go get github.com/Harry-Zhang121/pbimmutable
```

## How to Use

Import the library into your PocketBase `main.go` file (or wherever you configure your hooks):

```go
import (
	"log"
	"fmt"

	"github.com/Harry-Zhang121/pbimmutable" 
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	// ... other imports
)
```

The core of this library is the `MakeImmutable` function. You bind the function it returns to PocketBase's `OnRecordUpdate` event hook. `MakeImmutable` can accept a mix of string arguments (field names for immutability) and an optional single callback function of type `func(e *core.RecordEvent) error`.

There are several ways to use `MakeImmutable`:

### 1. Make Specific Fields Immutable (No Callback)

Provide a list of field names (as strings) that you want to make immutable.

```go
app.OnRecordUpdate("my_collection").Add(pbimmutable.MakeImmutable("field_one", "field_two"))

// Example: Make the 'email' field immutable for the 'users' collection
app.OnRecordUpdate("users").Add(pbimmutable.MakeImmutable("email"))
```
If an update attempts to change `field_one` or `field_two`, the operation will be rejected.

### 2. Make All User-Defined Fields Immutable (No Callback)

Call `MakeImmutable()` without any arguments. All user-defined fields in that collection will be treated as immutable. PocketBase system fields (like `id`, `created`, `updated`) are automatically excluded.

```go
app.OnRecordUpdate("documents").Add(pbimmutable.MakeImmutable())
```

### 3. Make Specific Fields Immutable AND Run a Custom Callback

Provide field names and a callback function. The callback will only run if the immutability checks for the specified fields pass.

```go
myCustomLogic := func(e *core.RecordEvent) error {
    log.Printf("Record %s in collection %s passed immutability checks. Running custom logic.", e.Record.Id, e.Record.Collection().Name)
    // This logic runs AFTER immutability checks pass, but BEFORE the database transaction is committed.
    // If this function returns an error, the entire transaction will be rolled back.
    // For example, to signal an issue based on other conditions:
    // if e.Record.GetString("status") == "finalized" && e.Record.GetFloat("balance") < 0 {
    //     return fmt.Errorf("custom validation failed: finalized record %s cannot have a negative balance", e.Record.Id)
    // }
    return nil
}

// Make 'important_data' immutable and then run myCustomLogic
app.OnRecordUpdate("sensitive_info").Add(pbimmutable.MakeImmutable("important_data", myCustomLogic))
```

### 4. Make All User-Defined Fields Immutable AND Run a Custom Callback

Provide only a callback function to `MakeImmutable()`. All user-defined fields will be treated as immutable, and the callback will run if those checks pass.

```go
app.OnRecordUpdate("legacy_records").Add(pbimmutable.MakeImmutable(myCustomLogic))
```

## How It Works

The `MakeImmutable` function processes its arguments (field names and an optional callback) and returns another function. This returned function conforms to the `func(e *core.RecordEvent) error` signature required by PocketBase's `OnRecordUpdate` hook.

When an update event occurs for a monitored collection:

1.  **Argument Parsing**: `MakeImmutable` first validates its own arguments. If you pass invalid arguments (e.g., two callbacks), an error is returned immediately when the hook runs.
2.  **Fetch Original Record**: The hook fetches the state of the record *before* the pending update to compare against.
3.  **Immutability Check**: It compares the values of the designated immutable fields (or all user-defined fields if none were specified) between the original record and the incoming data in `e.Record`.
4.  **Error on Change**: If any immutable field has been changed (and it's not a permitted system field like `updated`), the hook returns an `apis.NewBadRequestError`. This error prevents the update operation, and PocketBase will roll back the transaction.
5.  **Callback Execution**: If all immutability checks pass successfully:
    *   And if a user-defined callback function was provided to `MakeImmutable`, this callback is then executed.
    *   The callback receives the same `*core.RecordEvent`.
    *   **Crucially, the callback executes *after* the immutability checks have passed but *before* PocketBase attempts to commit the database transaction for the record update.**
    *   If the callback function returns an error, this error is propagated by the `MakeImmutable` hook. This will also cause PocketBase to roll back the entire transaction. The record will not be updated.
    *   If the callback function returns `nil` (no error), the `MakeImmutable` hook also returns `nil`.
6.  **Transaction Outcome**: If the `MakeImmutable` hook (including the result of the user callback, if any) returns `nil`, PocketBase proceeds with its internal operations. If no other subsequent hooks in the chain fail, PocketBase will then attempt to commit the database transaction for the record update.

System fields like `id`, `created`, and `updated` are generally allowed to change as they are managed by PocketBase. The `updated` field is explicitly allowed to change even if all fields are marked immutable. Other system fields are ignored by the "all fields immutable" logic.

## Error Handling

-   **Setup Errors**: If `MakeImmutable` is called with invalid arguments (e.g., multiple callbacks), an error is returned when the hook executes.
-   **Record Fetch Errors**: If the original record cannot be fetched for comparison, an error is returned, preventing the update.
-   **Immutability Violation**: If an immutable field is changed, a specific `apis.NewBadRequestError` is returned, indicating which field was modified.
-   **Callback Errors**: If the user-provided callback function returns an error, that error is propagated, leading to a transaction rollback.

## Example Scenario

Consider a `contracts` collection where `contract_terms` and `client_id` should never change after creation. Additionally, after confirming these are unchanged, you want to log the attempted update or perform another check.

```go
package main

import (
	"fmt"
	"log"

	"github.com/Harry-Zhang121/pbimmutable" // Replace Harry-Zhang121
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	// _ "your_project/migrations"
)

func main() {
	app := pocketbase.New()

	contractUpdateCallback := func(e *core.RecordEvent) error {
		log.Printf("Contract %s passed immutability checks for terms and client. Proceeding with update attempt.", e.Record.Id)
		// Example: further validation or action
		if e.Record.GetBool("is_archived") && !e.Record.GetBool("can_update_archived") {
			return fmt.Errorf("archived contract %s cannot be updated further without 'can_update_archived' flag", e.Record.Id)
		}
		return nil
	}

	// Make 'contract_terms' and 'client_id' immutable for 'contracts',
	// and run contractUpdateCallback if they are not changed.
	app.OnRecordUpdate("contracts").Add(pbimmutable.MakeImmutable("contract_terms", "client_id", contractUpdateCallback))

	// Optional: make all fields in 'audit_logs' immutable after creation, no callback
	app.OnRecordUpdate("audit_logs").Add(pbimmutable.MakeImmutable())

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

## Contributing

Feel free to open issues or submit pull requests if you find bugs or have suggestions for improvements.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
