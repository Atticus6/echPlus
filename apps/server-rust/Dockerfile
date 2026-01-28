FROM rust:1.83-alpine as builder

WORKDIR /app

RUN apk add --no-cache musl-dev

COPY Cargo.toml Cargo.lock ./
COPY src ./src

RUN cargo build --release

FROM alpine:latest

COPY --from=builder /app/target/release/echplus-server /usr/local/bin/

EXPOSE 3325

CMD ["echplus-server"]
