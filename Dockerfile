FROM ghcr.io/goreleaser/goreleaser-cross:latest

RUN apt-get update -y
RUN apt-get install -y libx11-dev
