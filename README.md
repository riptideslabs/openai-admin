
# openai-admin

Small CLI for OpenAI organization administration tasks.

## Requirements

- Go installed (see `go.mod` for the target version)
- Environment variables:
	- `OPENAI_API_KEY` (required)
	- `OPENAI_BASE_URL` (optional; override API base URL)
	- `OPENAI_ORG_ID` (optional)
	- `OPENAI_PROJECT_ID` (optional)

## Install / Run

Run directly:

```bash
go run . --help
```

Or build a binary:

```bash
go build -o openai-admin .
./openai-admin --help
```

## Commands

### Admin API keys

List all admin API keys (auto-paginates) with columnized output:

```bash
go run . admin-keys list
```

Create an admin API key (prints the one-time key value only):

```bash
go run . admin-keys create --name "Main Admin Key"
```

Create and copy the one-time key value to clipboard (macOS):

```bash
go run . admin-keys create --name "Main Admin Key" | pbcopy
```

Delete an admin API key by id:

```bash
go run . admin-keys delete key_abc
```

## Notes

- This CLI uses the `openai-go` SDK and relies on its environment-based configuration.
- Output is intended for human readability (tab-aligned columns).

## References

- OpenAI API Reference: https://platform.openai.com/docs/api-reference

