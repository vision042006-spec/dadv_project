.PHONY: install setup api worker frontend docker-start docker-stop docker-logs clean

install:
	cd frontend && npm install
	go mod download

setup:
	mkdir -p data/uploads

api:
	go run ./cmd/api/main.go

worker:
	python cmd/worker/worker.py

frontend:
	cd frontend && npm run dev

docker-start:
	docker-compose -f docker/docker-compose.yml up -d

docker-stop:
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f

docker-build:
	docker-compose -f docker/docker-compose.yml build

clean:
	rm -rf data/uploads/*
	rm -f data/dadv.db