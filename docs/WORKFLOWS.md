# Workflow Patterns

Use the App Store Connect workflow surfaces deliberately:

- `asc publish appstore`: canonical App Store upload and submission path
- `asc publish testflight`: canonical TestFlight upload and distribution path
- `asc workflow`: repo-specific orchestration of App Store Connect commands

Build and export the IPA with your project’s own reviewed build system. Then
hand the finished artifact to `asc`:

```bash
asc publish testflight \
  --app "$APP_ID" \
  --ipa "/path/to/App.ipa" \
  --group "$TESTFLIGHT_GROUP" \
  --wait \
  --output json
```

For an App Store submission:

```bash
asc validate --app "$APP_ID" --version "$VERSION"
asc publish appstore \
  --app "$APP_ID" \
  --ipa "/path/to/App.ipa" \
  --version "$VERSION" \
  --submit \
  --confirm \
  --output json
```

A workflow may compose public App Store Connect operations around that artifact:

```json
{
  "env": {
    "APP_ID": "1234567890",
    "TESTFLIGHT_GROUP": "Internal"
  },
  "workflows": {
    "testflight": {
      "description": "Upload an existing IPA and distribute it internally.",
      "steps": [
        {
          "name": "publish",
          "run": "asc publish testflight --app \"$APP_ID\" --ipa \"$IPA_PATH\" --group \"$TESTFLIGHT_GROUP\" --wait --output json",
          "outputs": {
            "BUILD_ID": "$.buildId",
            "BUILD_NUMBER": "$.buildNumber"
          }
        }
      ]
    }
  }
}
```

Run `asc workflow validate` before executing a workflow file. Workflow steps
are arbitrary shell commands, inherit the process environment, and should be
reviewed like source code.
