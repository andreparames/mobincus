# mobincus

Incus containers. Docker-compatible CLI. Use tools that speak Docker.

mobincus makes your Incus containers work with much of the Docker tool ecosystem — VS Code Dev Containers, CI runners, deployment scripts, monitoring agents, and other tools that call `docker` on the command line.

No Docker daemon. No changes to your workflow. Just drop mobincus on `PATH` as `docker` and Incus becomes visible to tools that expect a Docker host.

## What works today

`docker run`, `docker exec`, `docker ps`, `docker inspect`, `docker cp`, `docker create`, `docker start`, `docker wait`, `docker rm`, `docker volume`

VS Code Dev Containers can attach to Incus containers, exec into them, copy files, and inspect them — all through the standard Docker CLI interface.

## Status

Under development — successfully used with VS Code Dev Containers, but not production-ready.

---

*developed with OpenCode/DeepSeek*
