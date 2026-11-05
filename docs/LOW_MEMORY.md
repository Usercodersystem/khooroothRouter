# 128 MB RAM target

The service is intentionally tiny compared with a full Node.js/Next.js router, but no claim is made that a 128 MB VPS can reliably run Linux, Docker, this gateway, and several concurrent public streaming clients at once.

## Suggested initial settings
- `MAX_CONCURRENCY=1` or `2`
- `MAX_BODY_BYTES=262144` (256 KiB) if compatible with your API payloads
- Use a minimal OS and avoid Docker if the daemon's memory overhead is too high
- Keep TLS termination on another machine if possible
- Measure actual resident memory and test streaming, slow clients, and upstream failures

## Container memory
The compose memory limit is a guardrail. If the process exceeds it, the kernel may OOM-kill it. It is not a guarantee that the process will start or remain stable within the limit. The host OS and Docker daemon consume RAM outside the container's process budget.

## Streaming
The gateway uses Go's standard `httputil.ReverseProxy` rather than reading the entire upstream response into memory. It forwards streaming responses and flushes them promptly. Each open stream still consumes connection and goroutine resources, which is why concurrency limits matter.

## Before public deployment
This project currently has a single shared client API key and an in-memory per-IP limiter. It does not yet provide multi-user account management, persistent quotas, usage metering, or billing. Do not treat it as a production public SaaS gateway until those are implemented and tested.
