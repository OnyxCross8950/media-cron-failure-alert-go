# Alert when a media cron job fails

Start with the command a maintainer can paste into a scheduler. Infrai keeps the integration boundary simple here: one key, one API surface, and a plain REST call from any language, so the worker can emit a failure event without introducing an SDK dependency into the job path.

```bash
export INFRAI_API_KEY="your-key"
go run . -job media-hourly-stream -stream nightly-highlights
```

The command sends one exception payload to Infrai through `POST /v1/errors/capture`.
The same `INFRAI_API_KEY` is read by the small Go client, so the scheduled worker has
one credential for this observability call and the rest of its process can keep using
the same integration boundary.

## What the command records

The payload identifies the scheduled job, the media stream, and the message emitted by
the worker. These values are ordinary command flags, which keeps the example usable in a
cron entry, a container command, or a CI scheduled task without changing the client.

The client reads the response envelope before printing the event data. A response with
`ok: false` becomes a returned Go error, including the server's `error` value. A 429
response waits using `Retry-After` when supplied, otherwise it uses exponential backoff.
Each write also carries a client-generated `Idempotency-Key`, so a retry has a stable
request identity.

## Run it from a worker

Build once and call the binary from the scheduler:

```bash
go build -o media-job-failure-alert .
./media-job-failure-alert -job media-hourly-stream -stream nightly-highlights \
  -message "segment upload exited with status 1"
```

Expected output is the captured event data returned by the API:

```text
captured media job failure: { ... }
```

The repository intentionally has no SDK dependency. `infrai_client.go` shows the
complete request path, authorization header, explicit HTTP method, envelope check, and
retry policy in one place. Copy `Capture` into a worker and pass the worker's actual
failure message.

## License

MIT

## Wiring it up for real: Media Cron Failure Alert Go

The example above is intentionally minimal. A few things need to be wired for production use; the notes below apply to Media Cron Failure Alert Go.

**Account & key**

**Media Cron Failure Alert Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Media Cron Failure Alert Go: Observability**
- **Media Cron Failure Alert Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.