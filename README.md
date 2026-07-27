# mobincus

A Go CLI tool that translates Docker CLI commands into [Incus](https://linuxcontainers.org/incus/) API calls and returns output in Docker CLI format.

## Status

**Under development — not ready for production.**

mobincus implements enough of the Docker CLI surface to pass several official Docker CLI e2e tests and to connect VS Code Dev Containers to Incus containers.

## What works

```
docker version           docker inspect         docker exec
docker ps                docker create          docker start
docker run               docker rm              docker wait
docker cp                docker info            docker volume
docker ps --format       docker inspect --format
```

| Test | Status |
|---|---|
| `TestInspectInvalidReference` | ✅ |
| `TestTLSVerify` | ✅ |
| `TestRunAttach` (stdin/stdout/stderr) | ✅ |
| `TestRunInvalidEntrypointWithAutoremove` | ✅ |
| `TestProcessTermination` (signal forwarding) | ✅ |
| VS Code Dev Containers attach | ✅ |

## How it works

mobincus sits between your shell and the Incus daemon:

```
docker ps  →  mobincus  →  POST /1.0/instances
                         →  GET  /1.0/instances/<name>/state
                         →  POST /1.0/instances/<name>/exec (WebSocket)
```

It connects to Incus via the Unix socket at `/var/lib/incus/unix.socket` and translates Docker conventions (container IDs, labels, volumes) into Incus equivalents (instance names, `user.*` config keys, custom storage volumes).

## Project structure

```
mobincus/
├── bin/docker          → symlink to src/mobincus
├── docker-cli/          # cloned github.com/docker/cli
├── moby/                # cloned github.com/moby/moby
├── src/
│   ├── main.go
│   ├── cmd/             # cobra commands (run, ps, exec, inspect, cp, ...)
│   ├── incus/           # Incus REST API client + WebSocket exec
│   └── docker/          # output formatters + Go template helpers
└── AGENTS.md
```

## Quick start

```bash
export PATH="/path/to/mobincus/bin:$PATH"
docker ps
docker run --rm alpine echo hello
```

## Running tests

```bash
# mobincus own tests
cd src && go test ./tests/

# Docker CLI e2e tests
cd /path/to/docker-cli
cp vendor.mod go.mod
PATH="/path/to/mobincus/bin:$PATH" \
  TEST_DOCKER_HOST="unix:///var/lib/incus/unix.socket" \
  GOFLAGS=-mod=vendor go test -v -run TestName ./e2e/system/
```

---

*developed with OpenCode/DeepSeek*
