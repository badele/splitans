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

# build project
[group('golang')]
@go-fix:
  go mod tidy
  go mod vendor

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
  go test ./... -cover -coverprofile=coverage.out > /dev/null
  go tool cover -func=coverage.out
  go tool cover -html=coverage.out

# install precommit hooks
[group('precommit')]
@precommit-install:
  pre-commit install
  pre-commit install --hook-type commit-msg

# test precommit hooks
[group('precommit')]
precommit-test:
  pre-commit run --all-files

# Convert to neotex
[group('neotex')]
@neotex-from-ansi FILENAME:
    rm -f "{{ FILENAME }}.*"

    ./splitans -e cp437 -f ansi -E utf8 -F neotex "{{ FILENAME }}" -L 1000 > "{{ FILENAME }}.neop" 2>&1
    ./splitans -e utf8 -f neotex -E cp437 -F ansi "{{ FILENAME }}.neop" > "{{ FILENAME }}.neop.ans" 2>&1
    ansilove "{{ FILENAME }}" > /dev/null 2>&1
    ansilove "{{ FILENAME }}.neop.ans" > /dev/null 2>&1
    compare -metric SSIM "{{ FILENAME }}.png" "{{ FILENAME }}.neop.ans.png" "{{ FILENAME }}.diff.png" || touch "{{ FILENAME }}.error"

# Delete all neotex generated files
[group('neotex')]
@neotex-all-delete PATH:
    echo "Removing files computed ANSI files"
    find "{{ PATH }}" -name "*.ANS.*" -exec rm -f {} \; > /dev/null

# Convert to neotex
[group('neotex')]
neotex-all-from-ansi PATH:
    #!/usr/bin/env bash
    for file in $(find "{{ PATH }}" -name "*.ANS"); do
      if [ -e "${file}.neop.ans" ]; then continue; fi
      echo -e "\n\nConverting $file to neotex format"
      just neotex-from-ansi "$file" 2>&1 || exit 1
    done;

# Convert all ansi file to a single png
[group('neotex')]
ansi-all-to-one-png PATH:
    #!/usr/bin/env bash
    rm -f /tmp/one-ansi-file.ans-
    for file in $(find "{{ PATH }}" -name "*.ANS"); do
      echo -e "Converting $file to utf8 ansi format"
      ./splitans -e cp437 -f ansi -E cp437 -F ansi "$file" -L 1000 >> /tmp/one-ansi-file.ans 2>&1
    done;
    ansilove /tmp/one-ansi-file.ans

# Generate PNG comparison
[group('doc')]
@doc-comparison:
  curl -s https://16colo.rs/pack/1990/raw/WWANS157.ANS  | ./splitans  -e cp437 > /tmp/WWANS157.neo
  ./splitans -f neotex -F ansi /tmp/WWANS157.neo > /tmp/WWANS157.ans 2>&1
  reset
  cat /tmp/WWANS157.neo | pr -m -t -w 130
  ./splitans -f neotex -F ansi  /tmp/WWANS157.neo
  ./splitans -f neotex -F ansi -v /tmp/WWANS157.neo

# Previous README markdown
[group('doc')]
@doc-preview-markdown:
  godown
