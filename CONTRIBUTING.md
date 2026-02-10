# Contributing

Thanks for your interest in contributing to xelanote. This guide covers the workflow, coding
standards, and expectations for pull requests.

## Code of Conduct

By participating, you agree to follow the Code of Conduct in `CODE_OF_CONDUCT.md`.

## How to Contribute

1. Fork the repository and create a feature branch.
2. Make your changes with tests where appropriate.
3. Update `CHANGELOG.md` under `[Unreleased]` for user-facing changes.
4. Run the quality checks.
5. Open a pull request.

## Development Setup

```bash
make init
export JWT_SECRET="$(openssl rand -hex 32)"
make run-backend
make run-frontend
```

## Quality Checks

```bash
make quality
make test
make test-frontend
```

## Commit Messages

Use clear, conventional-style messages:

- `feat: add ...`
- `fix: correct ...`
- `docs: update ...`
- `chore: ...`

## Coding Standards

- Go: `gofmt` formatting and standard library patterns.
- Frontend: ESLint + Prettier rules in `frontend/eslint.config.js` and
  `frontend/prettier.config.cjs`.
- Keep changes focused. Avoid unrelated refactors.

## Reporting Security Issues

Please do not open public issues for security problems. Follow `SECURITY.md`.

