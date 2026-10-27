# ginmongo

Go + Gin + MongoDB behind nginx. The only third-party Go modules are Gin and
the official MongoDB driver; dependency injection is plain constructor
injection, no DI framework.

## Layout

    cmd/api/main.go                process entrypoint (config, logger, signals)
    internal/
      app/                         composition root: the only place that wires concrete types
      config/                      env-based config
      database/                    Mongo client lifecycle
      domain/                      entities, inputs, sentinel errors (no deps)
      repository/                  Mongo persistence, BSON docs stay private here
      service/                     business logic, declares the repository interface it needs
      handler/                     Gin handlers, declare the service interface they need
      middleware/                  request id, structured logging, panic recovery
      router/                      builds *gin.Engine from handlers
    deploy/nginx/nginx.conf        reverse proxy, rate limit, request id
    Dockerfile, docker-compose.yml nginx + api + mongo

## How the DI works

Each layer defines a small interface for what it needs from the layer below
("accept interfaces, return structs"), and `internal/app/app.go` passes the
concrete implementations in:

    config -> database.Mongo -> repository.UserRepository -> service.UserService
           -> handler.UserHandler -> router.New -> http.Server

* `service` never imports `repository` or the Mongo driver.
* `handler` never imports `service`; it only sees its own `UserService` interface.
* `HealthHandler` takes a `Pinger`, satisfied by `*database.Mongo`.
* Time is injected into the service as a `Clock`, so tests are deterministic.
* `internal/service/user_service_test.go` tests business logic with a hand-written fake repo.

To add a feature (e.g. `orders`): add `domain/order.go`, `repository/order_repository.go`,
`service/order_service.go`, `handler/order_handler.go`, then wire the four constructors
in `buildEngine` and pass the handler to `router.New`.

## Run

    cp .env.example .env     # change the password
    make init                # fetches gin + mongo driver, creates go.sum
    make up                  # nginx:80 -> api:8080 -> mongo (mongo is not published to the host)

Try it:

    curl -s localhost/readyz
    curl -s -X POST localhost/api/v1/users -H 'Content-Type: application/json' \
         -d '{"name":"Ada","email":"ada@example.com"}'
    curl -s 'localhost/api/v1/users?limit=10'
    curl -s -X PATCH localhost/api/v1/users/<id> -H 'Content-Type: application/json' -d '{"name":"Ada L."}'
    curl -s -X DELETE localhost/api/v1/users/<id>

Local dev without Docker for the API: `docker compose up -d mongo`, publish its port or
point `MONGO_URI` at a local mongod, then `make run`.

## Configuration (env)

| Var | Default |
|---|---|
| APP_ENV | development (`production` = JSON logs + Gin release mode) |
| HTTP_PORT | 8080 |
| MONGO_URI | mongodb://localhost:27017 |
| MONGO_DATABASE | ginmongo |
| TRUSTED_PROXIES | loopback + private ranges (so Gin trusts nginx's X-Forwarded-For) |
| HTTP_READ_TIMEOUT / HTTP_WRITE_TIMEOUT / HTTP_IDLE_TIMEOUT / HTTP_SHUTDOWN_TIMEOUT / MONGO_CONNECT_TIMEOUT | Go durations, e.g. `10s` |
