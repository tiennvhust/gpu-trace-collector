# Multi-stage build: full toolchain to compile, distroless to run.
# distroless/static = no shell, no libc, tiny attack surface — the right base
# for a static CGO_ENABLED=0 Go binary.
FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -o /out/collector ./cmd/collector

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/collector /collector
COPY configs/collector.yaml /etc/collector/collector.yaml
EXPOSE 4317 9464
ENTRYPOINT ["/collector", "-config", "/etc/collector/collector.yaml"]
