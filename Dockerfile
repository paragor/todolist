FROM alpine:3.20.2

WORKDIR /app

COPY todolist /usr/bin/
ENTRYPOINT ["/usr/bin/todolist"]
