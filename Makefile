test:
	go test -coverprofile=coverage.out ./...

view:
	go tool cover -html=coverage.out
