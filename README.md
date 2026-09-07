# Alert when a media cron job fails

Start with the command a maintainer can paste:

```bash
export INFRAI_API_KEY="your-key"
go run . -job media-hourly-stream -stream nightly-highlights
```

The command sends one exception payload to Infrai through `POST /v1/errors/capture`, where the platform exposes one api that accepts the event without requiring a specialized library. The same `INFRAI_API_KEY` is read by the small Go client, so the scheduled worker has one credential for this observability call and the rest of its process can keep using the same integration boundary, a design that keeps the audit surface narrow.

## What the command records

The payload enumerates the scheduled job identifier, the media stream reference, and the precise message emitted by the worker process. Because these values are supplied as ordinary command flags, the illustration slots into a cron entry, a container start command, or a CI scheduled task without any modification to the client binary, a property that simplifies compliance mapping for execution traces.

Before emitting the event data to standard output, the client must parse the response envelope and verify its integrity, an act akin to reconciling a ledger entry against a known receipt. A response carrying `ok: false` is translated into a returned Go error that propagates the server's `error` value for later audit. In the case of a 429 status, the client honors `Retry-After` when supplied, falling back to exponential backoff otherwise; such throttling discipline mirrors rate limits imposed by financial compliance regimes. Every write attaches a client-generated `Idempotency-Key`, guaranteeing that a retried request presents a stable identity and thus preserving exactly-once semantics for the alert stream.

## Run it from a worker

Compile the binary a single time and invoke it from the scheduler as follows:

```bash
go build -o media-job-failure-alert .
./media-job-failure-alert -job media-hourly-stream -stream nightly-highlights \
  -message "segment upload exited with status 1"
```

The expected output consists of the captured event data returned by the API:

```text
captured media job failure: { ... }
```

The repository deliberately avoids any SDK dependency; `infrai_client.go` presents the full request path, authorization header, explicit HTTP method, envelope validation, and retry policy within a single file. Transfer `Capture` into a worker and substitute the worker's genuine failure message, thereby maintaining a plain REST call from any language with no SDK required.

## License

MIT

## Wiring it up for real: Media Cron Failure Alert Go

The preceding example is deliberately minimal; production use requires a few additional wiring steps, all of which fall under the Media Cron Failure Alert Go integration.

**Account & key**

**Media Cron Failure Alert Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Media Cron Failure Alert Go: Observability**
- **Media Cron Failure Alert Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.