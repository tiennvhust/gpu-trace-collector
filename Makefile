# gpu-trace-collector — Makefile
#
# make tidy    resolve and pin dependencies (run once after cloning)
# make build   compile ./bin/collector
# make run     run locally against docker-compose Kafka (localhost:29092)
# make up      start Kafka + collector in docker compose
# make down    stop the stack
# make consume tail the telemetry topic (headers + keys) from the host

BIN := bin/collector

.PHONY: tidy build run up down consume clean

tidy:
	go mod tidy

build: tidy
	CGO_ENABLED=0 go build -trimpath -o $(BIN) ./cmd/collector

run: build
	./$(BIN) -config configs/collector.yaml

up:
	docker compose -f deploy/docker-compose.yml up --build -d

down:
	docker compose -f deploy/docker-compose.yml down

consume:
	docker compose -f deploy/docker-compose.yml exec kafka \
	  /opt/kafka/bin/kafka-console-consumer.sh \
	  --bootstrap-server localhost:9092 --topic telemetry.otlp \
	  --property print.key=true --property print.headers=true --from-beginning

clean:
	rm -rf bin
