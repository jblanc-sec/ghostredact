# Changelog

All notable changes to this project will be documented here.

## [v0.1.0] - 2025-08-21
### Added
- Initial release of GhostRedact CLI.
- Built-in detectors: email, phone, credit card (Luhn), IPv4/IPv6, IBAN.
- Modes: `mask` (default), `tag`, `hash` (with `-salt`).
- Input formats: text & JSON (recursive walk).
- Reporting: `--report` JSON counts per detector.
- Extensibility: `--custom` (JSON/YAML regex) for user-defined PII patterns.
- Optional locale pack: `-locale br` (CPF, CNPJ, CEP, RG).
