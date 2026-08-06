# Support

## Start Here

- Quick start and troubleshooting: [README.md](README.md)
- Questions and workflow help: [GitHub Discussions](https://github.com/dual1208/App-Store-Connect-CLI/discussions)
- Bugs and feature requests: [GitHub Issues](https://github.com/dual1208/App-Store-Connect-CLI/issues)

## Use Discussions For

- Install or upgrade help
- Authentication and private credential-file setup questions
- "How do I...?" workflow questions
- Automation, CI, or scripting advice
- Sharing examples, tips, and patterns with other users

## Use Issues For

- Reproducible bugs or regressions
- Incorrect help text, broken docs, or misleading output
- Clear feature requests for missing commands, flags, or workflows

## Useful Bug Report Checklist

Include as many of these as you can:

- `asc version`
- Your OS and shell
- The exact source commit used to build `asc`
- The exact command you ran
- Redacted stdout/stderr output
- Whether the issue still reproduces with `ASC_BYPASS_KEYCHAIN=1`
- Redacted `ASC_DEBUG=api` or `asc --api-debug ...` output when safe

## Common Gotchas

- the App Store Connect API can change underneath it
- Authentication resolves from private config files and complete environment credentials; `ASC_STRICT_AUTH=true` can help catch mixed sources
- Output defaults are TTY-aware: interactive terminals default to `table`, pipes and CI default to `json`
- Keep credential directories at mode `0700` and credential files at mode `0600`
