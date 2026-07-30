# Multi-stage build: full toolchain to compile, distroless to run.
# distroless/static = no shell, no libc, tiny attack surface — the right base
# for a static CGO_ENABLED=0 Go binary.
FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/collector /collector
# The private path's aggregators and client, in the same image. One image, several
# entrypoints — deploy/docker-compose.privacy.yml overrides `command` to pick one.
#
# » Deliberately NOT one image per role: the binaries are small and static, and a
# » single image means one build, one scan and one version to reason about. The two
# » aggregators are separate PROCESSES with separate configs and separate keys,
# » which is what the trust model requires; sharing a base image does not weaken it,
# » whereas sharing a process would.
COPY --from=build /out/dap-leader /dap-leader
COPY --from=build /out/dap-helper /dap-helper
COPY --from=build /out/prio-client /prio-client
COPY configs/collector.yaml /etc/collector/collector.yaml
EXPOSE 4317 9464 8080 8081 9465 9466
ENTRYPOINT ["/collector", "-config", "/etc/collector/collector.yaml"]
