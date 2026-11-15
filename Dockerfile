FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/etl ./cmd/etl

FROM gcr.io/distroless/static-debian13:nonroot AS api
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian13:nonroot AS etl
COPY --from=build /out/etl /etl
USER nonroot:nonroot
ENTRYPOINT ["/etl"]
