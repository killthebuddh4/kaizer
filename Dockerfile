FROM golang:1.23 AS build

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

WORKDIR /build

COPY ./ ./

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /main

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /main /main

EXPOSE 80

# Run
CMD ["/main"]