# Collaborator Guide

## Getting Started

```bash
git clone git@github.com:nhassl3/servicehub-backend.git
cd servicehub-backend
cp .env.example .env   # fill in secrets
make postgres redis    # local infra via Docker
make migrate-up        # apply migrations
make run               # start server
```

Dependencies: Go 1.26+, Docker, `make`, `migrate` CLI.

## Branch Strategy

| Branch | Purpose |
|---|---|
| `main` | Stable, production-ready. Protected — no direct pushes. |
| `bugfix/*` | Bug fixes. PR into `main`. |
| `feature/*` | New features. PR into `main`. |
| `chore/*` | Refactoring, CI, deps, docs. PR into `main`. |

Base every branch off `main` and keep it short-lived.

## Workflow

1. `git checkout -b bugfix/your-fix main`
2. Make changes, commit often with descriptive messages
3. `make lint && make test` — must pass
4. Push and open a PR against `main`
5. Request review from at least one maintainer
6. Squash-merge when approved

## Code Conventions

- **Language**: Go, follow [Effective Go](https://go.dev/doc/effective_go) and `gofumpt` formatting
- **Architecture**: Clean Architecture — domain → service → repository, never skip layers
- **Imports**: stdlib → third-party → internal, grouped by blank lines
- **Errors**: Wrap with `fmt.Errorf("context: %w", err)`, define sentinel errors in domain
- **No raw SQL**: all queries in `db/query/*.sql`, regenerate with `make sqlc`
- **No hand-editing** `internal/db/*.sql.go` — it's sqlc output
- **Mocks**: regenerate with `make mock` after changing domain interfaces
- **Tests**: table-driven with `testify`, use gomock, enable race detector
- **Secrets**: never commit `.env`, never log credentials or tokens

## Pull Request Checklist

- [ ] `make lint` passes
- [ ] `make test` passes (race-clean)
- [ ] `make vet` passes
- [ ] New code has tests
- [ ] Domain interfaces updated → `make mock` regenerated
- [ ] DB changes → new migration + `make sqlc` + test rollback
- [ ] Proto changes → PR in `servicehub-contracts` first

## Proto Contracts

Protobuf definitions live in the separate [`servicehub-contracts`](https://github.com/nhassl3/servicehub-contracts) repo (private Go module). If your change requires new/modified RPCs or messages:

1. PR in `servicehub-contracts` first
2. Tag a new version
3. Update `go.mod` in this repo to the new tag
4. Implement handler in `internal/transport/grpc/`

CI uses `GH_PAT` to authenticate with the private module — make sure your branch is compatible.

## Code Review

- Review for correctness, test coverage, error handling, and security
- Check that Redis DB selection matches intent (`cfg.Redis.DB` vs `cfg.Redis.DB+1`)
- Verify migrations are reversible (`make migrate-down`)
- Approve only when all checklist items are resolved

## Need Help?

Open a [Discussion](https://github.com/nhassl3/servicehub-backend/discussions) or tag a maintainer on your PR.
