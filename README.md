# mobincus

Incus containers. Docker-compatible CLI. Use tools that speak Docker.

mobincus makes your Incus containers work with much of the Docker tool ecosystem — VS Code Dev Containers, CI runners, deployment scripts, monitoring agents, and other tools that call `docker` on the command line.

No Docker daemon. No changes to your workflow. Just drop mobincus on `PATH` as `docker` and Incus becomes visible to tools that expect a Docker host.

## What works today

`docker run`, `docker exec`, `docker ps`, `docker inspect`, `docker cp`, `docker create`, `docker start`, `docker wait`, `docker rm`, `docker volume`

VS Code Dev Containers can attach to Incus containers, exec into them, copy files, and inspect them — all through the standard Docker CLI interface.

## Build and run tests

```bash
# Build
cd src && go build -o mobincus .

# Symlink so tools find it as docker
ln -sf "$PWD/mobincus" ../bin/docker

# Run mobincus's own tests
cd src && go test ./tests/

# Run Docker CLI e2e tests
cd /path/to/docker-cli
cp vendor.mod go.mod
PATH="/path/to/mobincus/bin:$PATH" \
  TEST_DOCKER_HOST="unix:///var/lib/incus/unix.socket" \
  GOFLAGS=-mod=vendor go test -v -run TestName ./e2e/system/
```

## Status

Under development — successfully used with VS Code Dev Containers, but not production-ready.

---

*developed with OpenCode/DeepSeek*
