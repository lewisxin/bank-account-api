FROM golang:1.27.4-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bank-account-api .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bank-account-api /bank-account-api

USER 65532:65532
ENV APP_HOST=:8080
EXPOSE 8080
ENTRYPOINT ["/bank-account-api"]
