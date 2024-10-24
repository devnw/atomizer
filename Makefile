all: build tidy lint fmt test

#-------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------
env=CGO_ENABLED=1

test: 
	CGO_ENABLED=1 go test -v -cover -failfast -race ./...

fuzz:
	@fuzzTime=$${FUZZ_TIME:-10}; \
	files=$$(grep -r --include='**_test.go' --files-with-matches 'func Fuzz' .); \
	for file in $$files; do \
		funcs=$$(grep -o 'func Fuzz\w*' $$file | sed 's/func //'); \
		for func in $$funcs; do \
			echo "Fuzzing $$func in $$file"; \
			parentDir=$$(dirname $$file); \
			go test $$parentDir -run=$$func -fuzz=$$func -fuzztime=$${fuzzTime}s; \
			if [ $$? -ne 0 ]; then \
				echo "Fuzzing $$func in $$file failed"; \
				exit 1; \
			fi; \
		done; \
	done

bench:
	go test -bench=. -benchmem ./...

lint: 
	golangci-lint run
	pre-commit run --all-files	

gomod2nix:
	gomod2nix generate

build: gomod2nix test
	$(env) go build ./...

release-dev: build-ci

upgrade:
	pre-commit autoupdate
	go get -u ./...

update:
	git submodule update --recursive

fmt:
	gofmt -s -w .

tidy: fmt
	go mod tidy

clean: 
	rm -rf dist
	rm -rf coverage

#-------------------------------------------------------------------------
# CI targets
#-------------------------------------------------------------------------
test-ci: 
	CGO_ENABLED=1 go test \
				-cover \
				-covermode=atomic \
				-coverprofile=coverage.txt \
				-failfast \
				-race ./...
	make fuzz FUZZ_TIME=10

build-ci: test-ci
	$(env) go build ./...

bench-ci: test-ci
	go test -bench=. ./... | tee output.txt

release-ci: build-ci	

#-------------------------------------------------------------------------
# Force targets
#-------------------------------------------------------------------------

FORCE: 

#-------------------------------------------------------------------------
# Phony targets
#-------------------------------------------------------------------------

.PHONY: build test lint fuzz
