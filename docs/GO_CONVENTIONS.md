# Go Coding Conventions

## Core Principles

- **Simplicity First:** Favor simple, readable code over clever solutions
- **Idiomatic Go:** Follow standard Go conventions and community practices
- **Standard Library:** Prefer the Go standard library over third-party dependencies

## Code Style

### General Guidelines

- Follow all conventions outlined in [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` to format all code before committing
- Keep functions small and focused on a single responsibility
- Create utility functions for any behavior that repeats more than twice
- Name variables clearly; avoid abbreviations unless universally understood (e.g., `i` for index)

### Naming Conventions

- **Packages:** Short, concise, lowercase, single-word names
- **Interfaces:** Use `-er` suffix for single-method interfaces (e.g., `Reader`, `Writer`)
- **Getters:** Omit `Get` prefix (use `Name()`, not `GetName()`)
- **Acronyms:** Keep consistent case (e.g., `userID`, `HTTPServer`, not `userId`, `HttpServer`)

## Dependency Management

- **Default to Standard Library:** Only introduce external dependencies when absolutely necessary
- **HTTP Routing:** Use `net/http` for all middleware and routing (no `mux`, `chi`, or `gin`) unless absolutely necessary. [Go 1.22+ has more advanced routing](https://go.dev/blog/routing-enhancements)
- **Evaluate Trade-offs:** Document why any external dependency is required

## API Design

- Follow RESTful principles for all API endpoints
- Use appropriate HTTP methods (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`)
- Return consistent JSON response structures
- Use proper HTTP status codes
- Version your APIs when breaking changes are necessary (e.g., `/api/v1/`)

## Error Handling

- Always check and handle errors explicitly
- Return errors rather than panicking (except in truly exceptional cases or when bootstrapping the system and dependencies aren't met)
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Don't ignore errors with blank identifier `_` without good reason

```go
// Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Avoid
_ = doSomething() // ignoring errors without justification
```

## Concurrency

- Use goroutines for concurrent operations, but don't overuse them
- Prefer channels for communication between goroutines
- Avoid shared mutable state; use channels or synchronization primitives (`sync.Mutex`) when necessary
- Always consider race conditions; run tests with `go test -race`
- Close channels when done writing to signal completion
- Use `context.Context` for cancellation and timeout control

## Security

- Validate and sanitize all user inputs
- Never log or expose sensitive data (passwords, tokens, PII)
- Use parameterized queries to prevent SQL injection
- Implement proper authentication and authorization
- Follow the principle of least privilege

## Logging

- Use `log/slog` for all structured logging
- Choose appropriate log levels:
  - **Debug:** Detailed information for diagnosing issues
  - **Info:** General informational messages
  - **Warn:** Warning messages for potentially harmful situations
  - **Error:** Error events that might still allow the application to continue
- Include relevant context in log messages (user ID, request ID, etc.)
- Never log sensitive information (passwords, tokens, credit cards)

```go
slog.Info("user authenticated",
    "user_id", userID,
    "ip_address", ipAddr,
)
```

## Testing

- Write unit tests for all new features and bug fixes
- All tests must pass before code can be merged
- Use table-driven tests for testing multiple scenarios

```go
func TestCalculate(t *testing.T) {
    tests := []struct {
        name    string
        input   int
        want    int
        wantErr bool
    }{
        {"positive number", 5, 25, false},
        {"zero", 0, 0, false},
        {"negative number", -5, 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Calculate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

- Test exported functions and methods
- Use `t.Helper()` in test helper functions
- Mock external dependencies (databases, APIs, etc.)
- Aim for meaningful test coverage, not just high percentages

## Documentation

### Exported Code

- Every exported function, type, method, and package must have a comment
- Start comments with the name of the element being documented
- Use complete sentences with proper punctuation

```go
// UserService handles all user-related operations.
type UserService struct {
    db *sql.DB
}

// GetUser retrieves a user by their ID from the database.
// It returns an error if the user is not found or if a database error occurs.
func (s *UserService) GetUser(id string) (*User, error) {
    // implementation
}
```

### Internal Comments

- Comment the **why**, not the **what**
- Explain business logic, non-obvious decisions, or workarounds
- Don't comment on what the code obviously does

```go
// Good: Explains WHY
// We retry 3 times because the payment gateway occasionally returns
// transient errors during high traffic periods
for i := 0; i < 3; i++ {
    err := processPayment()
}

// Bad: States the obvious WHAT
// Loop 3 times
for i := 0; i < 3; i++ {
    err := processPayment()
}
```

## Linting

- Use `golangci-lint` for all linting checks
- Fix **all** linting issues before committing code
- Configure project-specific linting rules in `.golangci.yml`
- Run linting as part of CI/CD pipeline

```bash
# Run locally before committing
golangci-lint run
```

---

**Remember:** These conventions exist to maintain code quality, readability, and team consistency. When in doubt, prioritize clarity and simplicity.
