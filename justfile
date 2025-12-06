#!/usr/bin/env just -f

# This help
@help:
    just -l -u

# Init go development
[group('golang')]
go-init:
  #!/usr/bin/env bash
  if [ ! -f go.mod ]; then
    go mod init github.com/badele/splitans
    go mod tidy
  fi

# format all go files
[group('golang')]
@go-fmt:
  go fmt ./...

# build project
[group('golang')]
@go-build: go-init
  go build

# Test project
[group('golang')]
@go-test:
  go test

# Compute code coverage
[group('golang')]
@go-coverage:
  go test ./tokenizer/... -cover -coverprofile=coverage.out
  go tool cover -func=coverage.out | grep -E "(tokenizer.go|token.go)" | grep -v "100.0%"

# install precommit hooks
[group('precommit')]
@precommit-install:
  pre-commit install
  pre-commit install --hook-type commit-msg

# test precommit hooks
[group('precommit')]
precommit-test:
  pre-commit run --all-files
