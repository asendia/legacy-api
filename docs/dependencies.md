# Dependency checks

Use Go 1.27.1. The container and Cloud Build test step use the same version. The scheduled function uses the supported `go127` runtime.

Keep `go.mod` and `go.sum` in the PR. Use the public Go module proxy and checksum database. Do not set `GOSUMDB=off`. Check release notes, source repositories, and release dates before a module update.

```sh
go mod verify
go vet ./...
go build -mod=readonly ./cmd
govulncheck ./...
```

Run `go test ./...` only with an isolated test database and empty mail credentials. The test setup deletes and creates tables. Do not point it at production.

The 2026-09-14 update uses pgx 5.10.0, OIDC 3.21.0, go-jose 4.1.5, and Mailjet 4.0.8. The old Mailjet v3 and x/crypto dependencies are no longer required. The selected module releases are more than seven days old. pgx 5.11.0 was less than seven days old and was not selected.

Tests cover stored message encryption, reminders, final delivery, retries, cancellation, account checks, and Telegram token checks. Tests cannot prove that no regression exists. Complete the live service checks in `telegram.md` before you enable Telegram.

The container must include `mail/template-*.html` for email delivery. These files are copied into the runtime image.

For rollback, use the previous application revision. Leave the additive Telegram tables in place. Keep the feature disabled until the live checks pass.
