.PHONY: test race vet run compare testbench matrixbench mergesortbench fmt

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

run:
	go run ./cmd/sim

compare:
	go run ./cmd/compare -trace all

testbench:
	go run ./cmd/testbench

matrixbench:
	go run ./cmd/matrixbench

mergesortbench:
	go run ./cmd/mergesortbench

fmt:
	gofmt -w ./cmd ./internal
