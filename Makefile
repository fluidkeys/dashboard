.PHONY: run
run:
	firejail --seccomp.enotsup=sendfile go run main.go

.PHONY: migrate
migrate:
	./migrations/migrate migrations/*.sql
