FROM golang:1.22.12 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags='-s -w' -o /out/fishingops-server ./cmd/server \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/fishingops-server /fishingops-server
COPY --from=build --chown=65532:65532 /out/data /data
ENV DATABASE_PATH=/data/fishingops.db
EXPOSE 8080
ENTRYPOINT ["/fishingops-server"]
