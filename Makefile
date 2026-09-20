.PHONY: help build web test test-db run dev-web docker lint

help:
	@echo "make build     - compila il frontend e il binario Go con la PWA embedded"
	@echo "make web       - compila solo il frontend (web/dist)"
	@echo "make test      - test Go (i test di integrazione richiedono SBRIGO_TEST_DATABASE_URL)"
	@echo "make run       - avvia il backend leggendo .env"
	@echo "make dev-web   - avvia Vite con proxy verso il backend su :8080"
	@echo "make docker    - build dell'immagine Docker"

web:
	cd web && npm ci && npm run build

build: web
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/sbrigo ./cmd/sbrigo

test:
	go vet ./... && go test ./...

run:
	set -a && . ./.env && set +a && go run ./cmd/sbrigo

dev-web:
	cd web && npm run dev

docker:
	docker build -t sbrigo:local .

lint:
	gofmt -l . && cd web && npx vue-tsc --noEmit -p tsconfig.json
