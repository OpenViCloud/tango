package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateDockerfile writes a Dockerfile into dir based on the detected stack.
// Returns an error if the stack has no template.
func GenerateDockerfile(dir string, stack DetectedStack) error {
	content, err := dockerfileTemplate(dir, stack)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0o644)
}

// detectGoVersion reads go.mod in dir and returns the major.minor version string
// (e.g. "1.26"), or "latest" if the file is missing or unparseable.
func detectGoVersion(dir string) string {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "latest"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Keep only major.minor (strip patch if present: "1.26.1" → "1.26")
				ver := parts[1]
				if dots := strings.Count(ver, "."); dots >= 2 {
					idx := strings.LastIndex(ver, ".")
					ver = ver[:idx]
				}
				return ver
			}
		}
	}
	return "latest"
}

func dockerfileTemplate(dir string, stack DetectedStack) (string, error) {
	switch stack {
	case StackGo:
		goVer := detectGoVersion(dir)
		goImage := "golang:" + goVer + "-alpine"
		if goVer == "latest" {
			goImage = "golang:latest"
		}
		return `FROM ` + goImage + ` AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app \
    $(go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./... | head -1)

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]
`, nil

	case StackNode:
		return `FROM node:24-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev
COPY . .

FROM node:24-alpine
WORKDIR /app
COPY --from=builder /app .
EXPOSE 3000
CMD ["node", "index.js"]
`, nil

	case StackPython:
		return `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt* pyproject.toml* ./
RUN pip install --no-cache-dir -r requirements.txt 2>/dev/null || pip install --no-cache-dir .
COPY . .
EXPOSE 8000
CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
`, nil

	case StackRust:
		return `FROM rust:1.77-alpine AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release && rm -rf src
COPY . .
RUN touch src/main.rs && cargo build --release

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/target/release/app .
EXPOSE 8080
CMD ["./app"]
`, nil

	case StackJava:
		return `FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline -q
COPY src ./src
RUN mvn package -DskipTests -q

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=builder /app/target/*.jar app.jar
EXPOSE 8080
CMD ["java", "-jar", "app.jar"]
`, nil

	case StackDotNet:
		return `FROM mcr.microsoft.com/dotnet/sdk:8.0 AS builder
WORKDIR /app
COPY *.csproj .
RUN dotnet restore
COPY . .
RUN dotnet publish -c Release -o out

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=builder /app/out .
EXPOSE 8080
CMD ["dotnet", "app.dll"]
`, nil

	default:
		return "", fmt.Errorf("no Dockerfile template for stack %q", stack)
	}
}
