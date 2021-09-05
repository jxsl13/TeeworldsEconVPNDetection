default: deploy

deploy:
	docker compose up -d --force-recreate --build

up: start

down: stop

start: build
	docker compose up -d

stop:
	docker compose down

build:
	docker compose build --force-rm

