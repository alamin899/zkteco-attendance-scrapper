# attendance-collector

A small **local Go application** that pulls **employees** and **attendance punches**
from ZKTeco fingerprint/face devices and stores them (deduplicated) in a database.
No web UI, no Python, no PHP — a single self-contained binary.

It talks the ZKTeco proprietary TCP protocol directly in pure Go (ported from the
proven `pyzk` logic, with strict packet framing so large logs — tens of thousands
of records — read reliably on high-volume devices).

## Features

- Pull **attendance** punches and store them with no duplicates (SHA-256 dedupe key).
- Sync **employees** (enrolled users: PIN, name, card, role).
- Auto-creates a placeholder employee for any unknown PIN so no punch is ever dropped.
- Two-device aware (an IN / entry device and an OUT / exit device).
- **SQLite** (default, zero-setup local file) or **MySQL** (share the DB with another app).
- `run` daemon polls on a **staggered** schedule so two devices never talk at once.
- Reconnect-with-retry on transient device errors (`TCP packet invalid`, timeouts).

## Requirements

- Go 1.24+ (only to build). The result is a single static-ish binary.
- Network line-of-sight to the device(s) on TCP port 4370.

## Setup

```bash
cp .env.example .env      # edit device IPs / DB settings
go build -o attendance-collector .
```

## Configuration (`.env`)

| Variable | Default | Meaning |
|---|---|---|
| `DB_DRIVER` | `sqlite` | `sqlite` or `mysql` |
| `DB_PATH` | `attendance.db` | SQLite file path |
| `DB_DSN` / `DB_HOST`… | – | MySQL connection (see `.env.example`) |
| `AUTO_MIGRATE` | `true` | Create tables if missing |
| `ZK_IN_IP` / `ZK_IN_PORT` | – | Entry device (a punch here = check-in) |
| `ZK_OUT_IP` / `ZK_OUT_PORT` | – | Exit device (a punch here = check-out) |
| `ZK_TIMEOUT` | `30` | Socket timeout (seconds) |
| `ZK_PASSWORD` | `0` | Device comm key |
| `ZK_RETRIES` | `2` | Extra reconnect attempts |
| `POLL_INTERVAL_MINUTES` | `10` | How often each device is polled by `run` (every 10 minutes) |
| `STAGGER_MINUTES` | `5` | Gap between device 1 and device 2 (device 2 pulls 5m after device 1) |

Leave one device's IP blank for a single-device setup.

## Usage

```bash
# One-shot commands
./attendance-collector test                     # connectivity (prints device time)
./attendance-collector sync-employees           # sync enrolled users -> employees
./attendance-collector pull                      # pull attendance -> attendances (dedup)
./attendance-collector pull --device=in          # only the IN device (in|out|ip|name)
./attendance-collector clear-device --device=out # store, then wipe the on-device log

# Daemon: poll forever on a staggered schedule (Ctrl+C to stop)
./attendance-collector run
```

The `--device` filter accepts `in`, `out`, a device IP, or part of its name.

### Keeping reads fast

Devices dump their **entire** log on every read, so a device holding tens of
thousands of records takes ~50s per pull. Run `clear-device` periodically (it
stores every punch in the DB first, then wipes the device) to keep reads instant.

## Database schema

Table/column names match the companion Laravel dashboard schema, so this collector
can write into that same database (set `DB_DRIVER=mysql` and point it at it):

- `devices` — the configured units (seeded automatically from `.env`).
- `employees` — enrolled people, unique by `device_user_pin`.
- `attendances` — raw punches, unique by `dedupe_hash = sha256(deviceId|pin|Y-m-d H:i:s)`.

## Docker Deployment

Build and run using the production multi-stage [Dockerfile](file:///Users/test/Documents/practice/attendance-collector/Dockerfile):

```bash
# 1. Build Docker image
docker build -t attendance-collector .

# 2. Run as a daemon container
docker run -d --name attendance-daemon --restart always --net=host --env-file .env attendance-collector run

# 3. View live logs
docker logs -f attendance-daemon
```

For complete step-by-step documentation for both Docker and Native setups, see [RUN_GUIDE.md](file:///Users/test/Documents/practice/attendance-collector/RUN_GUIDE.md).
For closed-source client delivery and binary installation instructions, see [CLIENT_DELIVERY_GUIDE.md](file:///Users/test/Documents/practice/attendance-collector/CLIENT_DELIVERY_GUIDE.md).

## Running as a service

Point a systemd unit / launchd plist / Docker container at `attendance-collector run`.
It logs to stdout and shuts down cleanly on SIGINT/SIGTERM.
