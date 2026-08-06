# App Store Connect CLI

`asc` is a scriptable Go CLI focused on App Store Connect: authentication,
apps, bundle IDs, signing, builds and uploads, TestFlight, users, metadata,
encryption declarations, Xcode helpers, and release workflows.

This is an independent, unofficial project and is not affiliated with or
endorsed by Apple.

## Install

Build a reviewed full commit from source:

```bash
git clone https://github.com/dual1208/App-Store-Connect-CLI.git
cd App-Store-Connect-CLI
git checkout "FULL_REVIEWED_COMMIT_SHA"
go build -trimpath -o asc .
install -m 0755 asc "$HOME/.local/bin/asc"
```

## Authenticate

Create an App Store Connect API key at
[App Store Connect](https://appstoreconnect.apple.com/access/integrations/api),
then register it:

```bash
asc auth login \
  --name "My Key" \
  --key-id "ABC123DEFG" \
  --issuer-id "12345678-abcd-1234-abcd-123456789012" \
  --private-key "/path/to/AuthKey_ABC123DEFG.p8" \
  --network

asc auth status --validate
asc auth doctor
```

For unattended machines, provide `ASC_KEY_ID`, `ASC_ISSUER_ID`, and
`ASC_PRIVATE_KEY_PATH` through the machine's secret manager.

## Common workflows

```bash
# Apps and bundle IDs
asc apps list --output table
asc web apps create --name "My App" --bundle-id "com.example.myapp" --sku "MYAPP"
asc bundle-ids list --output table

# Signing and capabilities
asc signing setup --app "APP_ID"
asc capabilities --help

# Build and upload
asc xcode archive --help
asc builds upload --app "APP_ID" --ipa "/path/to/MyApp.ipa"
asc builds wait --app "APP_ID" --build "BUILD_ID"

# TestFlight
asc testflight groups list --app "APP_ID"
asc testflight builds add-groups --build "BUILD_ID" --group "GROUP_ID"

# Encryption and release
asc encryption --help
asc validate --app "APP_ID" --version "1.0"
asc publish testflight --app "APP_ID" --ipa "/path/to/MyApp.ipa"
```

Run `asc --help` and `asc <command> --help` for the current command contract.
The generated command index is in [docs/COMMANDS.md](docs/COMMANDS.md).

## Privacy

The CLI has no telemetry, event spool, background worker, automatic update
check, advertising subsystem, issue-reporting command, or external
notification command. Network requests made by normal
commands are for Apple services selected by the requested App Store Connect
workflow. Explicit installation and documentation actions may access this
GitHub fork.

## Development

```bash
make format
make check-docs
make lint
make test
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and
[LICENSE](LICENSE).
