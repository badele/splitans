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

# Convert to neopack
[group('neotex')]
@neopack-from-ansi FILENAME:
    rm -f "{{ FILENAME }}.*"

    ./splitans -e cp437 -f ansi -E utf8 -F neopack "{{ FILENAME }}" -L 1000 > "{{ FILENAME }}.neop" 2>&1
    ./splitans -e utf8 -f neopack -E cp437 -F ansi "{{ FILENAME }}.neop" > "{{ FILENAME }}.neop.ans" 2>&1
    ansilove "{{ FILENAME }}" > /dev/null 2>&1
    ansilove "{{ FILENAME }}.neop.ans" > /dev/null 2>&1
    compare -metric SSIM "{{ FILENAME }}.png" "{{ FILENAME }}.neop.ans.png" "{{ FILENAME }}.diff.png" || touch "{{ FILENAME }}.error"

# Convert to neopack
[group('neotex')]
neopack-all-from-ansi PATH:
    #!/usr/bin/env bash
    for file in $(find "{{ PATH }}" -name "*.ANS"); do
      if [ -e "${file}.neop.ans" ]; then continue; fi
      echo -e "\n\nConverting $file to neopack format"
      just neopack-from-ansi "$file" 2>&1 || exit 1
    done;

# Delete all neopack generated files
[group('neotex')]
@neopack-all-delete PATH:
    echo "Removing files computed ANSI files"
    find "{{ PATH }}" -name "*.ANS.*" -exec rm -f {} \; > /dev/null
