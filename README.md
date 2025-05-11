# PocketBase Immutable Fields Plugin

[![Go Reference](https://pkg.go.dev/badge/github.com/Harry-Zhang121/pbimmutable.svg)](https://pkg.go.dev/github.com/Harry-Zhang121/pbimmutable)

This library provides a simple way to make specific record fields immutable in your [PocketBase](https://pocketbase.io/) application. Once a record is created, configured fields cannot be changed on update operations.

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

	"github.com/Harry-Zhang121/pbimmutable" // Import the library
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	// ... other imports
)
```

The core of this library is the `MakeImmutable` function. You bind the function it returns to PocketBase's `OnRecordUpdate` event hook.

There are two main ways to use `MakeImmutable`:

### 1. Make Specific Fields Immutable

Provide a list of field names (as strings) that you want to make immutable for a specific collection.

```go
// In your main function or where you set up PocketBase app
app := pocketbase.New()

// ... other app configurations

app.OnRecordBeforeUpdateRequest("my_collection").Add(func(e *core.RecordUpdateEvent) error {
    // Make 'field_one' and 'field_two' immutable for 'my_collection'
    // Note: The hook signature for OnRecordBeforeUpdateRequest is func(e *core.RecordUpdateEvent) error
    // We need to adapt the MakeImmutable to fit this or use a wrapper.
    // For simplicity with the current MakeImmutable, let's assume we are adapting its usage slightly
    // or that MakeImmutable is designed for a generic event type if possible.
    // However, the prompt specifically mentioned OnRecordUpdate.
    // Let's stick to OnRecordUpdate as per the prompt.
    return nil // Placeholder, actual binding below
})

// Correct usage with OnRecordUpdate as per prompt:
app.OnRecordUpdate("my_collection").Add(pbimmutable.MakeImmutable("field_one", "field_two"))

// Example: Make the 'email' field immutable for the 'users' collection
app.OnRecordUpdate("users").Add(pbimmutable.MakeImmutable("email"))

// Example: Make 'product_code' and 'initial_price' immutable for 'products' collection
app.OnRecordUpdate("products").Add(pbimmutable.MakeImmutable("product_code", "initial_price"))

// ... rest of your main function (app.Start(), etc.)
```

If an update operation attempts to change `field_one` or `field_two` in any record within `my_collection` after it has been created, the update will be rejected with a validation error.

### 2. Make All User-Defined Fields Immutable

If you call `MakeImmutable()` without any field name arguments, all user-defined fields in that collection will be treated as immutable. PocketBase system fields (like `id`, `created`, `updated`) are automatically excluded and will still be updated by PocketBase as needed.

```go
// In your main function or where you set up PocketBase app
app := pocketbase.New()

// ... other app configurations

// Make all user-defined fields in 'documents' collection immutable
app.OnRecordUpdate("documents").Add(pbimmutable.MakeImmutable())

// ... rest of your main function (app.Start(), etc.)
```

## How It Works

The `MakeImmutable` function returns another function that conforms to the `func(e *core.RecordEvent) error` signature required by PocketBase's `OnRecordUpdate` hook.

When an update event occurs for a monitored collection:
1. The hook fetches the state of the record *before* the pending update.
2. It compares the values of the designated immutable fields (or all fields if none were specified) between the original record and the incoming data in `e.Record`.
3. If any immutable field has been changed, the hook returns an `apis.NewBadRequestError`. This error prevents the update operation and PocketBase will typically display this error message in the Admin UI or return it in the API response.

System fields like `id`, `created`, and `updated` are always allowed to change as they are managed by PocketBase.

## Error Handling

- If the original record cannot be fetched, an error is returned, preventing the update.
- If an immutable field is changed, a specific `apis.NewBadRequestError` is returned, indicating which field was modified.

## Example Scenario

Consider a `contracts` collection where the `contract_terms` and `client_id` should never change after a contract is finalized (created).

```go
package main

import (
	"log"

	"github.com/Harry-Zhang121/pbimmutable"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	// uncomment to use example migrations
	// _ "your_project/migrations"
)

func main() {
	app := pocketbase.New()

	// Make 'contract_terms' and 'client_id' immutable for the 'contracts' collection
	app.OnRecordUpdate("contracts").Add(pbimmutable.MakeImmutable("contract_terms", "client_id"))

	// Optional: make all fields in 'audit_logs' immutable after creation
	app.OnRecordUpdate("audit_logs").Add(pbimmutable.MakeImmutable())

	// add migrate command
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

```

Now, if a user attempts to modify the `contract_terms` of an existing contract, they will receive an error, and the change will not be saved.

## Contributing

Feel free to open issues or submit pull requests if you find bugs or have suggestions for improvements.

## License

This project is licensed under the MIT License - see the LICENSE file for details (assuming MIT, you can add a LICENSE file).
