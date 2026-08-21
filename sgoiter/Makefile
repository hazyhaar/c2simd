export GOWORK=off
export PATH := /home/cl-ment/sdk/go1.27rc3/bin:/home/cl-ment/go/bin:$(PATH)
ROOT := $(abspath ..)
BIN  := $(ROOT)/bin

.PHONY: build test tribench smoke nightly probe-b3

build:
	go build -o $(BIN)/sgoiter ./cmd/sgoiter
	go build -o $(BIN)/probebench ./cmd/probebench
	go build -o $(BIN)/tribench ./tribench/cmd/tribench

test:
	go test ./... -count=1

tribench: build
	$(BIN)/tribench -root $(ROOT) -sgoiter $(BIN)/sgoiter -skip-ccgo -skip-bench

smoke: build
	./scripts/probe_smoke_ci.sh

nightly: build
	./scripts/probe_triangle_nightly.sh

probe-b3: build
	$(BIN)/probebench -root $(ROOT) -sgoiter $(BIN)/sgoiter -ccgo $$(command -v ccgo) \
		-work /tmp/sgoiter_probebench -out ./spec/bench/probe_microopt -skip-io -repeat 3
	$(BIN)/probebench -validate ./spec/bench/probe_microopt/probe_report.json -require-ccgo
