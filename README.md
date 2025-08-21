# GhostRedact — PII redaction CLI (MVP)

Fast, single-binary CLI that redacts common PII in **text** or **JSON**.
Defaults: emails, phone numbers, credit cards (Luhn), IPv4/IPv6, IBAN.

## Build (Linux/WSL2)

```bash
go mod tidy
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/ghostredact ./cmd/ghostredact
# Windows .exe
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/ghostredact.exe ./cmd/ghostredact
```

## Usage

```bash
ghostredact -in sample.txt -out clean.txt
ghostredact -in sample.json -out clean.json -format json

# Choose strategy
ghostredact -in sample.txt -mode hash -salt "yoursecret"

# Limit to certain types (overrides defaults and locales)
ghostredact -in sample.txt -types email,cc,phone

# Optional counts report (JSON)
ghostredact -in sample.txt -out clean.txt -report report.json
ghostredact -in sample.txt -report -    # write JSON report to stderr
```

### Custom user patterns (JSON or YAML)
Let users add *their own* PII patterns without rebuilding:

`patterns.json`:
```json
{
  "patterns": [
    {"name": "SSN", "regex": "\\b\\d{3}-\\d{2}-\\d{4}\\b"},
    {"name": "UUID", "regex": "\\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}\\b"}
  ]
}
```

`patterns.yaml`:
```yaml
patterns:
  - name: SSN
    regex: "\b\d{3}-\d{2}-\d{4}\b"
  - name: UUID
    regex: "\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}\b"
```

Run with:
```bash
ghostredact -in sample.txt -out clean.txt --custom patterns.json
# or
ghostredact -in sample.txt --custom patterns.yaml -report -
```

**How custom rules redact:**  
- `-mode tag` → `<NAME>` (e.g., `<SSN>`).  
- `-mode hash` → `<NAME:hash>` (short SHA-256 of the original).  
- `-mode mask` → also `<NAME>` (generic masking for arbitrary patterns).

### Flags (MVP)
- `-in`       : input file path (default: stdin)
- `-out`      : output file path (default: stdout)
- `-format`   : `text` (default) or `json` (walks values & redacts strings)
- `-mode`     : `mask` (default), `hash`, or `tag`
- `-salt`     : optional salt for `hash` mode
- `-types`    : comma list of detectors to enable (blank = defaults; overrides -locale)
- `-locale`   : optional locale pack (e.g., 'br')
- `-custom`   : JSON/YAML file with user-defined regex patterns
- `-report`   : write JSON counts report to file, or `-` for stderr
- `-threads`  : worker goroutines for text mode (default: CPU count)

MIT licensed. No telemetry.

---

For locale-specific detectors (e.g. Brazil: CPF, CNPJ, CEP, RG) see `docs/LOCALES.md`.
