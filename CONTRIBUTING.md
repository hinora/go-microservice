# Contributing

## Test coverage requirement

All new code must include automated tests. Changes should preserve 100% test coverage for the affected package and should add integration tests when behavior crosses service, broker, gateway, transporter, or discovery boundaries.

Before submitting changes, run:

```bash
GOTOOLCHAIN=local go test ./... -cover
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./...
```
