# Personal website

This is where I will eventually have the source code for my own little place on the interwebs.

## Blog

Blog posts can be written in markdown and stored in the `./blog` directory. The blog posts can have
the following frontmatter:

```
---
title: "<title>"
date: <date>
tags:
 - <tag 1>
 - <tag 2>
 - ...
---
```

or in the following format:

```
---
title: "<title>"
date: <date>
tags: [<tag 1>, <tag 2>, ...]
---
```

## Admin UI

A Sveltia CMS (a lightweight, Decap/Netlify-CMS-compatible headless CMS) instance is served at
`/admin/` for editing `blog` and `notes` posts through the browser instead of by hand. It commits
directly to the `fbegyn/website` GitHub repo via the GitHub API — there's no server-side CMS logic,
just two static files (`cmd/server/static/admin/index.html` and `config.yml`).

The route is disabled by default and only registered when both `--admin-user`/`--admin-pass` (or
the `WEBSITE_ADMIN_USER`/`WEBSITE_ADMIN_PASS` env vars) are set, gating it behind HTTP Basic Auth
since the page and `config.yml` are otherwise publicly reachable and reveal the repo name.

Sign-in to the CMS itself uses a GitHub personal access token (fine-grained, scoped to
`fbegyn/website` with `contents: read/write`) via Sveltia's "Sign In with Token" option — no OAuth
app or extra infra needed.

The Go server reads `./blog`/`./notes`/`./talks` off disk at startup and on `SIGHUP`
(`systemctl reload website` in production), so a post published through the CMS shows up after
the next reload, without a full restart.

`talks` is intentionally not a CMS collection — it's a separate git submodule (`fbegyn/talks`), and
a single Sveltia config can only target one GitHub repo.

### Local development

`config.yml` sets `local_backend: true`, which lets Sveltia skip GitHub auth entirely and write
straight to your local working copy when accessed via `localhost`. To use it:

1. Run `npx decap-server` from the repo root (a local proxy that reads/writes the working copy).
2. Run the site with admin auth enabled, e.g. `go run ./cmd/server serve --admin-user=test --admin-pass=test`.
3. Open `http://localhost:3114/admin/` — Sveltia detects the local backend automatically.

## Deploying

```
website-deploy <user>@<host> <root directory>
```

Syncs `./blog`, `./notes`, `./templates`, `./talks` and `./cmd/server/static` to `<root>/blog`,
`<root>/notes`, `<root>/templates`, `<root>/talks` and `<root>/static` over rsync/ssh. The `talks`
submodule is checked out first; the script aborts if it is still empty. Afterwards it reloads the server on the host with
`sudo systemctl reload website` (override with `WEBSITE_RELOAD_CMD`).

Available as `website:deploy` in the dev shell, `nix run .#website-deploy -- <user>@<host> <root>`,
or `make deploy DEPLOY_TARGET=<user>@<host> DEPLOY_ROOT=<root>`.
