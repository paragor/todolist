build_local:
	goreleaser release --snapshot --clean

install:
	CGO_ENABLED=0 go build -o /usr/local/bin/todolist main.go
	/usr/local/bin/todolist config-persist

update_default_config:
	rm config_example.yaml || true
	TODOLIST_HOME=./ go run main.go config-persist --config=config_example.yaml
