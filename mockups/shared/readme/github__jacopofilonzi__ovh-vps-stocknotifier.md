# OVH VPS Stock Notifier

Watches the stock of OVHcloud VPS plans and notifies you when the plans you care about become
available (or run out again) in the datacenters you choose.

## Quick start (Docker)

```sh
mkdir ovh-vps-stocknotifier && cd ovh-vps-stocknotifier
# copy docker-compose.yaml from this repository here, then:
mkdir data
docker compose up -d                      # start the scraper: it waits for a configuration
docker compose run --rm notifier tui      # configure it: it starts checking within seconds
docker compose logs -f                    # follow what it does
```

The image is private as long as the repository is: see
[Deploying from GHCR](#deploying-from-ghcr-private-repository) to log in first.

On Linux, `data/` must be writable by the container user (uid 1000):
`sudo chown 1000:1000 data` if you created it as another user. The scraper tells you if it isn't.

## What it watches

- **Stock** of each selected plan, per datacenter and per operating system (Linux and/or
  Windows: OVH tracks them separately).
- **Orderability**: whether a plan is still on sale (shown in OVH's order funnel), so you know
  when a plan you watch is withdrawn, or comes back.

Notifications are sent only when something changes, never on every check.

## Supported subsidiaries

Prices, currency and VAT follow the OVH subsidiary you choose:
IT, FR, DE, ES, GB, IE, NL, PL, PT, MA, SN, TN, CA, QC, AU, SG, IN, ASIA, WE, WS, US.

The US subsidiary has its own plan codes and datacenters.

## Configuration (TUI)

The TUI writes `config.json`; the scraper picks up every change within a few seconds, no restart
needed.

- **First run**: a setup wizard asks, in order, for the subsidiary, datacenters, operating
  systems, plans, notifiers (optional) and check interval. Nothing is saved until you confirm
  at the end, so the scraper never starts from a half-done configuration.
- **Afterwards**: a settings menu, with a summary of the configuration and of the scraper's
  status. Every confirmed change is saved right away.
- **Plans** are listed with vCore, RAM and monthly price, and only those on sale in the selected
  datacenters. A watched plan that was withdrawn stays listed as `(withdrawn)`.
- **Notifiers** can be tested with a plain test message, or with a *preview with live data*: the
  current stock of your plans, fetched right now (marked `[TEST]`, without touching the stored
  state). *Send current stock to all enabled notifiers* in the main menu does the same in one
  step.
- **Check now** asks the running scraper for an immediate check (see
  [How it works](#how-it-works)) and shows what it found. If the scraper isn't running, it
  offers a read-only check instead: the live stock is shown, but nothing is notified or saved.
- Changing the subsidiary or the datacenters removes, after asking, the choices that no longer
  apply.

