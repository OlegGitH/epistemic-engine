FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY apps/control-plane/go.mod apps/control-plane/go.sum* ./apps/control-plane/
RUN cd apps/control-plane && go mod download
COPY . .
WORKDIR /src/apps/control-plane
RUN CGO_ENABLED=0 go build -o /out/control-plane ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/control-plane /control-plane
EXPOSE 8080
ENTRYPOINT ["/control-plane"]
