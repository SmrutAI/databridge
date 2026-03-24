# syntax=docker/dockerfile:1
#
# Build context must be the smritea-cloud root:
#   docker build -f databridge/Dockerfile --target <target> .
#
# go.work replace directives inside databridge/ reference ../conveyor
# and ../smritea-sdk/go.  We therefore copy all three sibling trees into /src/
# so that /src/databridge/go.work can resolve ../conveyor → /src/conveyor.

# ── build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-bookworm AS builder

WORKDIR /src

# Copy sibling dependency trees first (referenced by go.work replace directives)
COPY conveyor/       ./conveyor/
COPY smritea-sdk/go/ ./smritea-sdk/go/

# Copy the pipeline source tree; go.work lives here and its replace directives
# now resolve correctly: ../conveyor → /src/conveyor, ../smritea-sdk/go → /src/smritea-sdk/go
COPY databridge/ ./databridge/

WORKDIR /src/databridge

RUN go mod download

RUN CGO_ENABLED=1 go build -o /out/bootstrap ./cmd/lambda/
RUN CGO_ENABLED=1 go build -o /out/server    ./cmd/server/
RUN CGO_ENABLED=1 go build -o /out/codewatch ./cmd/codewatch/

# ── lambda-intake ─────────────────────────────────────────────────────────────
FROM public.ecr.aws/lambda/provided:al2023-amd64 AS lambda-intake
COPY --from=builder /out/bootstrap ${LAMBDA_TASK_ROOT}/bootstrap
CMD ["bootstrap"]

# ── azure-functions ──────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS azure-functions
COPY --from=builder /out/server /app/server
COPY databridge/host.json /app/host.json
COPY databridge/api/      /app/api/
WORKDIR /app
CMD ["/app/server"]

# ── standalone ────────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS standalone
COPY --from=builder /out/server /app/server
WORKDIR /app
EXPOSE 8080
CMD ["/app/server"]
