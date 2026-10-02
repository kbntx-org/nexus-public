# AGENTS.md — Nexus Monorepo

Global conventions. Documentation rules live in [`docs/AGENTS.md`](docs/AGENTS.md).

## Repository Structure

- `apps/`: product applications.
- `platform/core`: cluster-level building blocks (networking, certificates, secrets, ingress,
  operators).
- `platform/services`: workloads hosted on the platform that are not product apps.

`mise.toml` at the root is the only source of runtime and tool versions; never hardcode them
elsewhere.

## Architecture Standards

Everything of the same kind must look the same: two NestJS backends, two Go backends or two charts
share one structure, whatever the language or framework. Copy the closest existing example
(`apps/smelt` is the reference) rather than inventing a new shape.

- **Backends are a modular monolith**, organized by domain, never by technical layer. A domain owns
  its model, repository, service, controller and errors. Domains call each other only through
  services, never through another domain's repository or data. Only tiny, domain-agnostic helpers
  live outside a domain.
- **Frontends are domain-driven per page.** A page owns its root component, services, context,
  models and components. Pages never import from each other; reuse goes through `shared`.
- **Helm:** one file per resource kind in `templates/`; environment overrides in
  `values.<env>.yaml`; third-party software goes through an umbrella chart.

## Naming and Comments

- No abbreviations or acronyms in identifiers, no single-letter variables (except Go receivers,
  `i`/`j` in `sort.Slice`, `err`, `ctx`), no magic numbers or strings.
- **The default is no comment.** Comment only a special case the code cannot express: a non-obvious
  _why_, a hidden constraint, or a workaround.
- TypeScript: one source of truth per closed set of values; define an `enum` and derive types and
  runtime lists from it.
- Go: interfaces live in the package that uses them.

## Dependencies

pnpm only. After any `package.json` change run `pnpm i --no-frozen-lockfile` and commit the
lockfile. Before adding any npm package or Go module, check it is still maintained; archived or
abandoned projects are not acceptable.

## Checks Before Finishing

**Mandatory, never skip, even for a one-line or docs-only change.** Before every commit, amend or
push, run the same formatting checks as CI against `main`, which cover only what changed:

```sh
pnpm nx format:check --base main --libs-and-apps
pnpm nx format:check --base main
pnpm nx affected -t format-check test
```

Both `format:check` runs are needed: `--libs-and-apps` covers files that belong to an nx project,
the plain run covers the rest of the git diff (root `AGENTS.md`, for example). A formatting failure
in CI means this check was skipped. Linting is left to CI. Update the docs for any feature you add,
change or remove.

## Safety

Never read or edit gitignored files that may hold secrets or local config (`.env*`, `*.tfvars`,
`vars*.yaml`, `environment.json`, and anything similar): use their `.example` counterpart. Never run
`terraform` (`plan`, `apply`, `destroy`) and never touch a live cluster.

## Tracking Work

Features and bugs are tracked as GitHub issues and shipped as stacked PRs (`gh stack`). Stack only
changes that depend on each other; an unrelated change gets its own branch and PR from `main`, so
PRs are never coupled. The user opens the PRs; never open, push or submit one unless asked.

- **Kicking off a feature or bug:** create the issue with `gh issue create --body-file`. `gh` can't
  render the issue forms in `.github/ISSUE_TEMPLATE/`, so use the matching form's field labels as
  `##` headings and fill each with concrete content (goal, scope with real paths, verifiable
  acceptance criteria). Pick labels from `gh label list`; never invent one. Give the user the URL.
- **Assignee, project and status:** every issue and PR is assigned to the user (`--assignee @me` or
  `gh issue edit` / `gh pr edit --add-assignee @me`). Only issues, not PRs, go in the `Nexus`
  project (`gh project item-add 1 --owner kbntx-org --url <url>`), with Status set to `Backlog` or
  `In Progress` (`gh project item-edit`). `gh project` needs the `read:project` and `project` token
  scopes (`gh auth refresh -s read:project,project`).
- **Size:** one commit per PR, and keep the PR small. Fold follow-up changes into that commit
  (amend) instead of adding commits.
- **Linking:** every PR mentions its issue in the description (`Closes #<n>` or `Part of #<n>`).
- **Writing a PR description:** when asked, read the PR with `gh pr view` and `gh pr diff` (in a
  stack, describe only that PR's slice), fill in
  [`.github/pull_request_template.md`](.github/pull_request_template.md) (Why / What / Notes), link
  the issue with `Closes #<n>` (or `Part of #<n>`), and apply it with `gh pr edit --body-file`.

## Keeping This File Current

Update this file when a convention is established or corrected (`docs/AGENTS.md` for documentation
rules). Add only what the code can't show. Each `AGENTS.md` has a sibling `CLAUDE.md` containing
only `@AGENTS.md`; never put rules in `CLAUDE.md`.
