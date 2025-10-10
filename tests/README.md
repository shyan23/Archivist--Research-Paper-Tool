# Tests Directory

This directory contains all test files for the Archivist project.

## Structure

```
tests/
├── unit/                  # Unit tests for individual components
│   ├── analyzer/         # Analyzer component tests
│   ├── compiler/         # LaTeX compiler tests
│   ├── generator/        # LaTeX generator tests
│   ├── parser/           # PDF parser tests
│   ├── storage/          # Metadata storage tests
│   ├── worker/           # Worker pool tests
│   └── fileutil/         # File utility tests
├── integration/          # Integration tests for end-to-end workflows
├── helpers/              # Test helper functions and utilities
│   └── testhelpers/      # Shared test helper code
├── scripts/              # Test execution scripts
│   ├── test.sh          # Main test runner
│   └── test-docker.sh   # Docker-based test runner
└── testdata/            # Test data files
    ├── sample_pdfs/     # Sample PDF files for testing
    └── expected_outputs/ # Expected output files for validation

## Running Tests

### Run all tests
```bash
go test ./tests/...
```

### Run unit tests only
```bash
go test ./tests/unit/...
```

### Run integration tests only
```bash
go test ./tests/integration/...
```

### Run tests with coverage
```bash
go test -cover ./tests/...
```

### Run tests using scripts
```bash
# Local tests
./tests/scripts/test.sh

# Docker-based tests
./tests/scripts/test-docker.sh
```

## Writing Tests

### Unit Tests
- Place in appropriate subdirectory under `tests/unit/`
- Use package name matching the component being tested
- Focus on testing individual functions/methods in isolation
- Use mocks for external dependencies

### Integration Tests
- Place in `tests/integration/`
- Use package `integration_test`
- Test complete workflows across multiple components
- May use real dependencies where appropriate

### Test Helpers
- Place in `tests/helpers/testhelpers/`
- Shared utilities, mocks, and fixtures
- Import as `archivist/tests/helpers/testhelpers`

## Test Data
- Sample PDFs go in `tests/testdata/sample_pdfs/`
- Expected outputs go in `tests/testdata/expected_outputs/`
- Use `t.TempDir()` for temporary test files