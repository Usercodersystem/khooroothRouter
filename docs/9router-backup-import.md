# 9Router backup compatibility

Current 9Router backups use SQLite at `db/data.sqlite` inside the data directory. The importer accepts that database directly or inside ZIP/TAR.GZ archives. Legacy JSON exports with top-level arrays such as `providerConnections`, `apiKeys`, and `combos` are also recognized.

The importer checks SQLite integrity, maps recognizable provider connections/API keys/combos/aliases, and stores every imported table row in `legacy_data` so unrecognized data is not silently discarded. It recognizes provider connection fields such as `provider`, `authType`, `name`, `email`, `priority`, `isActive`, and JSON `data`. Common API-key provider base URLs are inferred when absent. Provider-specific OAuth refresh, custom headers, and protocol translation are not migrated in this milestone. Access tokens may be retained as bearer credentials but are not refreshed.

Keep the original backup until all migrated providers and keys have been verified. See [the upstream 9Router backup documentation](https://github.com/decolua/9router/blob/master/gitbook/content/en/deployment/localhost.md).
