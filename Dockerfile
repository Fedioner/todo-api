FROM golang:1.26.4 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/todo-api ./cmd/todo-api
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/todo-api ./cmd/todo-api

FROM gcr.io/distroless/static-debian12:nonroot
ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
COPY --from=build /out/todo-api /todo-api
EXPOSE 8080
ENTRYPOINT ["/todo-api"]
