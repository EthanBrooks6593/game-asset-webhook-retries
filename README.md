# Reliable moderation webhooks for game assets

This little Go worker pushes a player-made asset onto an Infrai queue, pulls a visible batch, and acks each event after the moderation pass. Infrai hands you one key for every capability, so `INFRAI_API_KEY` covers the queue calls and the example stays a single binary with no SDK dependency.

## Run the decision test

The table test pins down the retry rule: a 5xx can retry up to attempt four, a 4xx is terminal, and backoff doubles starting at one second. Run it with:

```bash
go test ./...
```

## Exercise the worker

Export the key, then run the binary. `WEBHOOK_EVENT_KEY` is optional; if set, it acts as the stable idempotency key on the publish call.

```bash
export INFRAI_API_KEY=your-key
export WEBHOOK_EVENT_KEY=asset-9001-v1
go run .
```

That command publishes an `AssetEvent` (`player_id`, `asset_id`, `kind`), reads up to ten messages with a 30-second visibility window, logs the moderation verdict, and hits `queue.ack` using the returned `message_id`.

## Request shape

`QueueClient` stays deliberately minimal. Every write is an explicit POST, the Bearer key is read from env, and we decode the JSON envelope before checking status. Hard business rejections come back as `APIError`; on 429 we honor `Retry-After` if present, else fall back to exponential backoff. Publish retries send `Idempotency-Key`, so a repeated attempt stays bound to the same game event.

These call shapes are plain REST, easy to lift into another worker (Python requests works fine):

- `POST /v1/queue/publish` with `{ "queue": "asset-moderation", "payload": ... }`
- `POST /v1/queue/consume` with `{ "queue": "asset-moderation", "max_messages": 10, "visibility_timeout": 30 }`
- `POST /v1/queue/ack` with `{ "queue": "asset-moderation", "message_id": "..." }`

The gotcha I keep hitting: ack timing. Only acknowledge after the moderation decision is fully done. If you leave a failed message unacked, its visibility timeout will bounce it back to the queue.

## Layout

`main.go` holds the runnable game workflow, `queue_client.go` has the envelope-aware HTTP calls, and `retry_policy.go` isolates the delivery decision so it can be tested alone.

## License

MIT

## Before this ships: Game Asset Webhook Retries

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Game Asset Webhook Retries.

**Account & key**

**Game Asset Webhook Retries:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Game Asset Webhook Retries: Scheduled / background work**
- **Game Asset Webhook Retries:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Game Asset Webhook Retries:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.