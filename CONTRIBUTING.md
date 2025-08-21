# Contributing to GhostRedact

Thanks for considering a contribution!

## How to contribute
1. **Open an issue** describing the change or bug.
2. **Fork** the repo, create a branch: `feat/your-change` or `fix/your-bug`.
3. Keep changes focused and small; add tests when possible.
4. Run `go vet` and ensure the CLI builds.
5. Open a PR with a clear description and usage notes.

## Scope & philosophy
- Core detectors stay **international by default** (email, phone, cc with Luhn, IPv4/IPv6, IBAN).
- Region-specific IDs go behind `-locale` or **user-provided** `--custom` regex.
- Avoid heavy dependencies; keep the single-binary UX.

## Code style
- Go 1.22, `go fmt` and `go vet`.
- Keep CLI flags stable; document breaking changes in `CHANGELOG.md`.

## License
By contributing, you agree your contributions are licensed under the MIT License.
