test:
	go test ./tests/unit
	go test

benchmark:
	go test -bench=. ./tests/benchmark
