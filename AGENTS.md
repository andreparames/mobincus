# mobincus — Docker-compatible CLI for Incus

## Purpose

A Go CLI tool that accepts the same arguments as the Docker CLI, translates them into calls to the [Incus REST API](https://linuxcontainers.org/incus/docs/main/api/), and returns output in Docker CLI format.

The goal is to make the [Docker CLI e2e tests](https://github.com/docker/cli/tree/master/e2e) pass using this tool.

## Project structure

```
mobincus/
├── AGENTS.md
├── bin/docker          → symlink to src/mobincus (for running e2e tests)
├── docker-cli/          # cloned from github.com/docker/cli
├── src/
│   ├── main.go          # entry point
│   ├── go.mod / go.sum
│   ├── cmd/
│   │   ├── root.go          # cobra root, StatusError, Execute()
│   │   ├── container.go     # docker container subcommand tree
│   │   ├── container_ls.go  # docker container ls
│   │   ├── inspect.go       # docker inspect
│   │   ├── ps.go            # docker ps
│   │   ├── run.go           # docker run
│   │   └── version.go       # docker version
│   ├── incus/
│   │   ├── client.go        # Incus REST API client (unix socket)
│   │   ├── exec.go          # WebSocket exec streaming
│   │   └── types.go         # shared types
│   └── docker/
│       └── format.go        # Docker-compatible output formatters
```

## Development approach

1. Pick the simplest failing Docker CLI e2e test.
2. Implement the minimum `mobincus` functionality to make it pass.
3. Move to the next test.
4. Each command should mirror Docker's output format, exit codes, and error messages.

## Key learnings

### Running Docker CLI e2e tests

- Tests live in `docker-cli/e2e/`, organized by subcommand.
- Tests use `gotest.tools/v3/icmd` to run `docker <args>` as a subprocess.
- `icmd.RunCmd(icmd.Command("docker", "ps"))` — hardcodes the binary name as `"docker"`.
- `result.Assert(t, icmd.Expected{Out, Err, ExitCode})` — checks output with `strings.Contains`.
- All tests call `environment.Setup()` in `TestMain`, which **requires** `TEST_DOCKER_HOST` to be set. It sets `DOCKER_HOST` from this value.
- `environment.Setup()` does NOT check daemon reachability — it only sets env vars.
- The repo uses `vendor.mod` instead of `go.mod`. To run tests, copy/rename it:
  ```bash
  cp vendor.mod go.mod
  GOFLAGS=-mod=vendor go test -v -run TestName ./e2e/system/
  ```
- The repo must be at `$GOPATH/src/github.com/docker/cli/` (or use a symlink) for `GO111MODULE=auto` mode.
- Prepend `bin/` to `PATH` so the test framework finds our `docker` symlink:
  ```bash
  PATH="/path/to/mobincus/bin:$PATH" TEST_DOCKER_HOST="unix:///var/lib/incus/unix.socket" go test ...
  ```

### Our binary requirements

- Binary must be named `docker` on `$PATH` (symlink from `bin/docker` → `src/mobincus`).
- Must set `SilenceErrors: true` and `SilenceUsage: true` on cobra root to avoid Docker-incompatible prefix output.
- Use `StatusError` type (with `StatusCode` and `Status`) in `Execute()` to control exit codes and stderr output.
- Must accept `--host` / `-H` and `DOCKER_HOST` env var (even if unused) for compatibility with tests that pass `-H`.
- For `docker run`, use `cmd.Flags().SetInterspersed(false)` so flags after the image name (like `-c` in `sh -c`) aren't consumed by cobra.
- Use `os.Exit(exitCode)` directly in `run` command rather than returning StatusError, because cobra may mangle the exit code.

### Incus API

- Default connection: Unix socket at `/var/lib/incus/unix.socket`.
- All API responses are wrapped in a standard envelope:
  ```json
  {"type":"sync","status":"Success","status_code":200,"metadata":{...}}
  ```
- `GET /1.0` — server info. Key fields are nested under `environment`:
  ```go
  type ServerEnvironment struct {
      Server        string `json:"server"`
      ServerVersion string `json:"server_version"`
      OSName        string `json:"os_name"`
      KernelVersion string `json:"kernel_version"`
      Driver        string `json:"driver"`
      DriverVersion string `json:"driver_version"`
      ...
  }
  ```
- `GET /1.0/instances` — list of instance URLs (not `/1.0/containers`).
- `GET /1.0/instances/<name>` — instance details.
- `POST /1.0/instances` — create an instance (async operation).
- `PUT /1.0/instances/<name>/state` — start/stop an instance.
- `POST /1.0/instances/<name>/exec` — execute a command (async, requires WebSocket).
- `DELETE /1.0/instances/<name>` — delete an instance.
- `GET /1.0/operations/<id>/wait` — wait for an async operation to complete.
- Async operations return `{"type": "async", "operation": "/1.0/operations/<id>"}`.
- URLs in responses already include `/1.0/` prefix — strip it before passing to `Client.get()` / `Client.request()`.
- Exec in non-interactive mode with `wait-for-websocket: true` allocates 3 FDs (0=stdin, 1=stdout, 2=stderr) + control; all 3 must be connected before the command starts.
- WebSocket connections: use `gorilla/websocket` with a custom `NetDial` that connects to the Unix socket.

### Matching Docker error output

- `docker inspect FooBar` → stdout `[]`, stderr `error: no such object: FooBar`, exit 1.
- `docker: 'docker version' accepts no arguments` — error for `version` with args.
- `docker version` → stdout starts with `Client:\n Version:`.
- `docker --version` / `docker -v` → stdout contains `Docker version`.

## Current status

- [x] Project scaffolding (Go, Cobra, directory structure)
- [x] `incus version` command working
- [x] `docker inspect` implemented (returns error for unknown objects)
- [x] `docker ps` — list Incus instances in Docker-like format
- [x] `--tlsverify` flag handling with Docker-compatible errors
- [x] `docker run` with `-a`, `--rm`, image lookup, WebSocket exec streaming
- [x] `TestInspectInvalidReference` passes
- [x] `TestTLSVerify` passes
- [x] `TestRunAttach` (stdin, stdout, stderr) passes
- [x] `TestRunInvalidEntrypointWithAutoremove` passes
- [x] `TestProcessTermination` (signal forwarding via control WebSocket) passes
- [x] `docker create` command
- [x] `docker start` command
- [x] `docker rm` command (with `-f` force)
- [x] `docker wait` command
- [x] `docker run -d` detached mode
- [x] `docker create` command
- [x] `docker start` command
- [x] `docker rm` command (with `-f` force)
- [x] `docker wait` command
- [x] `docker cp` implementation (file API, recursive dirs, tar stdout, follow-link)
- [x] `version --format` with Go template `json` function support

## Next steps (in rough order)

- [ ] `--version` / `-v` flags on root command
- [ ] `-H` / `--host` / `DOCKER_HOST` flag support
- [ ] `docker container ls` — alias for `docker ps`
- [ ] `docker inspect` JSON output for real containers
- [ ] Run more e2e tests: `TestTCPSchemeUsesHTTPProxyEnv`, `TestCliPluginsVersion`, `TestGlobalArgsOnlyParsedOnce`
