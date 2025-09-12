# AGENTS.md for Taskman

## Build, Lint, and Test Commands
- **Build**: `go build .`
- **Lint**: `golangci-lint run`
- **Test**: `go test ./...`
- **Run a Single Test**: `go test -run TestFunctionName` (replace `TestFunctionName` with the actual test name)

## Code Style Guidelines
- **Imports**: Group standard library, third-party, and local imports separately. Use blank lines to separate these groups.
- **Formatting**: Use `gofmt` for formatting code. Ensure consistent indentation and spacing.
- **Types**: Prefer using concrete types over interfaces unless necessary. Use structs for data structures.
- **Naming Conventions**: Use camelCase for variables and functions; PascalCase for types and structs. Avoid abbreviations.
- **Error Handling**: Always check for errors and handle them appropriately. Use `fmt.Errorf` for wrapping errors with context.
- **Comments**: Write clear and concise comments for complex logic. Use full sentences and proper punctuation.

## Cursor Rules
- Ensure all code adheres to the defined cursor rules in `.cursor/rules/`.

## Copilot Rules
- Follow the guidelines specified in `.github/copilot-instructions.md` for using Copilot effectively.