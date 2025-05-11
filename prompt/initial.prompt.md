Make a pocketbase plugin library to let user easily set attributes as immutable. 

## Background
Immutable constrain is missing in the pocketbase database. But it is an important feature to enable secure data handling. But we can achieve similar result bu using pocketbase event hook.

## How users use our library
Users should be able to import our library using `import()` and go download it from github.
Our library should provide a function to be bound to `app.OnRecordUpdate` hook. There are three ways for user to use our function. For the examples below let' assume our function is called `MakeImmutable()`.

- `app.OnRecordUpdate("<table-name>").BindFunc(MakeImmutable(e *core.RecordEvent)) //Make all attributes immutable`
- `app.OnRecordUpdate("<table-name>").BindFunc(MakeImmutable(e *core.RecordEvent, )) //Make all attributes immutable`

## How to implement

Below is an example code to check if the `amount` field is changed in the `topup` table. 
```go
// --- Hook for topup cancellation ---
	// Uses OnRecordUpdate. If a topup is cancelled, we need to revert the balance.
	app.OnRecordUpdate("topup").BindFunc(func(e *core.RecordEvent) error {
		log.Printf("Topup Update Hook: Triggered for record %s\n", e.Record.Id)

		// Fetch the record state *before* the update
		originalRecord, err := e.App.FindRecordById("topup", e.Record.Id) // Changed collection to "topup"
		if err != nil {
			log.Printf("Topup BeforeUpdate Hook: Error fetching original record %s: %v\n", e.Record.Id, err) // Changed "purchase" to "topup"
			// Returning error prevents the update
			return fmt.Errorf("failed to fetch original topup record %s: %w", e.Record.Id, err) // Changed "purchase" to "topup"
		}

		// Check if 'amount' field is being changed
		originalAmount := originalRecord.GetFloat("amount")
		pendingAmount := e.Record.GetFloat("amount") // e.Record holds incoming data merged with original before validation/save

		// Direct float comparison (use epsilon comparison if precision issues arise)
		if originalAmount != pendingAmount {
			log.Printf("Topup BeforeUpdate Hook: Attempt to modify immutable 'amount' field on record %s (Original: %.2f, Attempted: %.2f). Rejecting update.\n", // Changed "Purchase" to "Topup"
				e.Record.Id, originalAmount, pendingAmount)
			// Return an error to reject the entire update operation
			customMessage := "Validation Error: The topup amount cannot be changed after the record is created."
			return apis.NewBadRequestError(customMessage, map[string]any{
				// Optional: add structured data (may or may not be displayed in UI, but useful for API clients)
				"field":    "amount",
				"reason":   "immutable",
				"recordId": e.Record.Id,
			})
		}
```

Because `BindFunc` only takes functions with `func(e T) error` signiture we need to implement a function to return this function type. This is the core of our library.
`func MakeImmutable(e *core.RecordEvent, immutableFieldNames []string) func(e *core.RecordEvent) error`

In this function it should check if the required field has been changed. If so return a coresponding error. This error will be also visiable on web dashboard.


## Your task
I want you to:
1. Implement this library. Covering edge cases and include error handling.
2. Create all necessary files to make it a go module.
3. Write a documentation in a README.md file to explain how users can use our library.
