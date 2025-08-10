test:
	go test -coverprofile=coverage.txt ./...

view:
	go tool cover -html=coverage.txt
