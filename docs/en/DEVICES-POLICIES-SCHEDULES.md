# Device inventory, policies, and schedules

## Device inventory

Open **Devices**, choose a client, and request its inventory. On Windows, the agent queries present Plug and Play devices and reports manufacturer/provider/version and problem codes. Code 28 is called out as a missing driver; other non-zero codes are reported as device problems. On Linux, the agent enumerates PCI/USB devices and checks for a driver binding in `/sys`.

The inventory does not download or install drivers. A missing sysfs binding can be expected for some USB, virtual, or generic devices, so treat the status as a diagnostic hint rather than proof that a driver must be installed.

## Group policies

The **Policies & schedule** page creates a declarative installation policy from packages already uploaded to Master, with an optional OS filter and an installation mode. In this first iteration, the OS selector is a case-insensitive substring match against the client's OS field, not a full group/tag system.

## Scheduler

Schedules use an interval from 5 minutes to 30 days. When due, Master selects **online** clients matching the OS filter and dispatches the package installation. The scheduler checks for due work about every 30 seconds. If no client matches or dispatch fails, the schedule is still advanced to its next interval; check Master logs for errors.

> **Warning:** each schedule firing repeats the selected package installation. Test with a test client and package first. Use only trusted and idempotent installers for production deployments.

## Database migration

For an existing database, apply only the new migration; do not replay the complete migration sequence over an existing schema:

```sh
make migrate-apply NAME=007_policies_schedules
```

For a fresh database, Docker applies this migration when initializing the volume for the first time. Adding a file under `docker-entrypoint-initdb.d` does not re-run it for an existing `postgres_data` volume.
