build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o todolist main.go

install:
	CGO_ENABLED=0 go build -o /usr/local/bin/todolist main.go
