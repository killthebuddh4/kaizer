FROM golang:1.23 AS build

# Set destination for COPY
WORKDIR /build

COPY ./ ./

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /main

FROM scratch

COPY --from=build /main /main

EXPOSE 80

# Run
CMD ["/main"]