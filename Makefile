.PHONY: run
run:
	go run main.go

.PHONY: migrate
migrate:
	./migrations/migrate migrations/*.sql
