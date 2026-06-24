# syntax=docker/dockerfile:1
#
# Build the image with a context two levels above this Dockerfile, so the sibling module
# checkouts referenced by the go.work replace directives are available to COPY. The
# Makefile invokes it as: docker build -f Dockerfile --target <target> ../..

# ── build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-bookworm AS builder

WORKDIR /src

# Copy the sibling dependency trees first (referenced by go.work replace directives).
COPY golang/conveyor/        ./golang/conveyor/
COPY polyglot/smritea-sdk/go/ ./polyglot/smritea-sdk/go/

# Copy the pipeline source tree; go.work lives here and its replace directives
# resolve against the sibling trees copied above.
COPY golang/databridge/ ./golang/databridge/

WORKDIR /src/golang/databridge

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
COPY golang/databridge/host.json /app/host.json
COPY golang/databridge/api/      /app/api/
WORKDIR /app
CMD ["/app/server"]

# ── standalone ────────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS standalone
COPY --from=builder /out/server /app/server
WORKDIR /app
EXPOSE 8080
CMD ["/app/server"]
