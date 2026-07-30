# gpu-trace-collector — Makefile
#
# make tidy    resolve and pin dependencies (run once after cloning)
# make build   compile ./bin/collector
# make run     run locally against docker-compose Kafka (localhost:29092)
# make up      start Kafka + collector in docker compose
# make down    stop the stack
# make consume tail the telemetry topic (headers + keys) from the host
#
# Private path (Project A — see docs/PRIVACY.md):
# make test         run the whole suite; red is your remaining exercise list
# make test-dp      week 1: differential-privacy mechanisms and accounting
# make test-field   week 2: prime field arithmetic — start here
# make test-vdaf    week 2: XOF + Prio3, including the draft test vectors
# make test-dap     week 3: DAP codec, aggregators, end-to-end
# make vectors      just the draft test vectors, verbosely (the week-2 gate)
# make fuzz         fuzz the DAP codec for 60s
# make bench        benchmarks that feed docs/BENCHMARKS.md
# make up-privacy   Kafka + collector + DAP leader + helper
# make keys         generate demo HPKE and verify keys into .env.privacy

BIN := bin/collector

.PHONY: tidy build run up down consume clean \
        test test-dp test-field test-vdaf test-dap vectors fuzz bench \
        up-privacy down-privacy keys

tidy:
	go mod tidy

build: tidy
	CGO_ENABLED=0 go build -trimpath -o bin/ ./cmd/...

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

# ── the private path ─────────────────────────────────────────────────────────
#
# `make test` is red until the exercises are done, and that is the intended
# workflow: the failures ARE the task list, in dependency order. Work bottom-up —
# field, then xof, then prio3, then dap — because a failure in a lower layer makes
# every layer above it fail for reasons that have nothing to do with its own code.

test:
	go test ./...

test-dp:
	go test -race ./internal/dp/...

test-field:
	go test ./internal/vdaf/field/...

test-vdaf:
	go test ./internal/vdaf/...

test-dap:
	go test -race ./internal/dap/... ./internal/privacy/...

# The week-2 gate. Green here is the single most credible artifact in the project:
# objective proof the spec was implemented correctly. Put the output in the README.
vectors:
	go test -v -run TestDraftTestVectors ./internal/vdaf/prio3

# The DAP codec parses attacker-controlled bytes on a public endpoint, which makes
# it the best fuzzing target in the repository. Expect a finding in the first minute.
fuzz:
	go test -fuzz=FuzzDecodeReport -fuzztime=60s ./internal/dap

# Feeds docs/BENCHMARKS.md — "quantify the privacy tax".
bench:
	go test -bench=. -benchmem -run='^$$' ./internal/vdaf/... ./internal/dap/...

up-privacy:
	docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.privacy.yml up --build -d

down-privacy:
	docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.privacy.yml down

# Demo keys for local runs only. Real deployments load these from a secret store —
# a verify key in a file lets any client forge a valid proof.
keys:
	@echo "# Generated for local development only. Do not commit." > .env.privacy
	@echo "DAP_HELPER_TOKEN=$$(head -c 16 /dev/urandom | xxd -p)" >> .env.privacy
	@echo "DAP_VERIFY_KEY_GPU_UTIL=$$(head -c 16 /dev/urandom | xxd -p)" >> .env.privacy
	@echo "DAP_VERIFY_KEY_GPU_ERRORS=$$(head -c 16 /dev/urandom | xxd -p)" >> .env.privacy
	@echo "wrote .env.privacy — HPKE keypairs need« EXERCISE 41»; add them here once Seal works"
