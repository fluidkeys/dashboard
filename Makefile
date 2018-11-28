.PHONY: run
run:
	go run main.go

.PHONY: migrate
migrate:
	./migrations/migrate migrations/*.sql

.PHONY: migrate_heroku
migrate_heroku:
	MIGRATE_HEROKU=1 ./migrations/migrate migrations/*.sql
