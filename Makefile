test:
	go test ./tests/unit

benchmark:
	go test -bench=. ./tests/benchmark
