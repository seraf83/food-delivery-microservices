FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/delivery-service ./cmd/delivery-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/delivery-service /delivery-service
EXPOSE 80
ENTRYPOINT ["/delivery-service"]
CMD ["serve"]
