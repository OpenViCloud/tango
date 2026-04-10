# ── Stage 1: Build FE (runs once, platform-independent) ──
FROM --platform=$BUILDPLATFORM node:24-alpine AS fe-builder
RUN npm install -g pnpm
WORKDIR /web
COPY web/package.json web/pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm build
# Vite build output ra /web/dist

# ── Stage 2: Build Go binary (per-platform) ─────
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go-builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy FE build vào đúng chỗ để go:embed nhúng vào binary
COPY --from=fe-builder /web/dist ./cmd/api/static
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o tango ./cmd/api

# ── Stage 3: Final image ──────────────────────────
# Includes git (for cloning repos) and buildctl (for talking to buildkitd)
FROM moby/buildkit:latest AS buildkit-bins

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata git openssh-client sshpass \
    python3 py3-pip && \
    pip install --no-cache-dir --break-system-packages ansible
COPY --from=buildkit-bins /usr/bin/buildctl /usr/local/bin/buildctl
WORKDIR /app
COPY --from=go-builder /app/tango .
COPY migrations ./migrations
EXPOSE 8080
CMD ["./tango"]
