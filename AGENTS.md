# AGENTS.md

## Build, Lint, and Test Commands
- **Build the project**: 
  ```bash
  go build ./...
  ```
- **Lint the code**: 
  ```bash
  golangci-lint run
  ```
- **Run all tests**: 
  ```bash
  go test ./...
  ```
- **Run a single test**: 
  ```bash
  go test -run TestFunctionName ./path/to/testfile.go
  ```

## Code Style Guidelines
- **Imports**: Group standard library imports, third-party imports, and local imports separately.
- **Formatting**: Use `gofmt` for formatting code. Ensure consistent indentation and spacing.
- **Naming Conventions**: 
  - Use CamelCase for types and exported functions.
  - Use lowerCamelCase for variables and unexported functions.
- **Error Handling**: Always check for errors and handle them appropriately. Use `fmt.Errorf` for wrapping errors.

## Cursor Rules
- (Include any rules found in .cursor/rules/)

## Copilot Rules
- (Include any rules found in .github/copilot-instructions.md)