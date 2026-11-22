# Instructions for coding agents

This repository is **webapp-bulma** — the Bulma theme for the
`github.com/the-maldridge/webapp` framework, shipped as its own Go module. It is
the worked example of a theme that lives *outside* the framework: templates, a
vendored Bulma bundle, two hand-written assets, and a `register.go`. Nothing here
is a runnable service; there is no `main` except the vendored fetcher.

The framework never imports this package. An **application** blank-imports it, which
registers the theme under the name `bulma` during `init`; `WithTheme("bulma")` or
`<PREFIX>_THEME=bulma` then selects it. See `README.md` for the consumer-facing story.

## Commands

```bash
make all       # fmt, vet, build, test  (the default gate)
make test      # go test -count=1 ./...   — fast (~0.5s), no network
make race      # go test -race -count=1 ./...
make check     # go run ./internal/fetchbulma -check   — offline bundle verify
make generate  # go generate .  — FETCHES the Bulma release in bulma.lock (network)
```

No `CGO_ENABLED=1` is needed here (unlike the framework, whose tests open SQLite).
The Bulma bundle is **committed**, so a fresh clone builds and tests with no
network and no generate step. Never make the build depend on `make generate`.

Go directive is `go 1.26.0` in `go.mod`. `bulma_test.go` is an **external** test
package (`package bulma_test`), so it can only use the exported surface: `FS()`,
`Static()`, `Required`.

## The `go.work` trap

A `go.work` is present locally and is **gitignored** (see `.gitignore`). It is:

```
use (
	.
	../webapp
	../webapp/examples
)
```

While it exists it **supersedes** the published `require webapp v0.0.1` and resolves
the framework from the sibling `../webapp` checkout. Consequences an agent hits fast:

- In a clone **without** the siblings, every `go` command fails with "directory not
  found". Fix: delete `go.work` (and `go.work.sum`), or run single commands with
  `GOWORK=off` to resolve the framework from its published version instead.
- Edits you make to `../webapp` are picked up here **immediately** while the
  workspace exists. A test that passes here may be passing against unreleased
  framework code — re-check with `GOWORK=off` before trusting it.
- It is gitignored precisely because committing it would name a directory that only
  exists when both repos are cloned side by side.

## Layout

| Path | The thing it is |
| --- | --- |
| `p2/` | The theme's templates. `FS()` mounts this directory as `@core/…`. |
| `static/css/bulma.min.css` | Vendored Bulma bundle, committed, **locked** — do not hand-edit. |
| `static/css/LICENSE` | Bulma's MIT licence, committed, **locked** — travels with the binary. |
| `static/css/theme.css` | The theme's **own** stylesheet (sticky footer, login centering, burger contrast, the light/dark `.switch`). Hand-edit by design. |
| `static/js/navbar.js` | The mobile navbar toggle. Hand-edit by design; must keep referencing `wa-nav-toggle`. |
| `static/js/theme.js` | The light/dark toggle: sets `data-theme` on `<html>`, persists the choice in `localStorage` key `wa-theme`, applied before paint. Loaded synchronously from `base.p2`'s head (no `defer`) so there is no flash. Pairs with the `#wa-theme-toggle` switch in `fragments/nav.p2`. |
| `bulma.lock` | Which Bulma release is vendored + the sha256 of every locked file. |
| `internal/fetchbulma` | Generator that advances the bundle; `-check` verifies offline. |
| `register.go` | **The only file that imports the framework.** Everything else is data. |

`bulma.go` (doc, embeds, `FS()`/`Static()`) deliberately imports only `embed` and
`io/fs`. Keeping the framework import isolated to `register.go` is intentional — do
not pull `webapp` into other files.

## The static URL prefix is `/static/core/`

`Static()` returns `static/` rooted, and the framework serves it under
`/static/core/…` (namespace `core`, from `themetest.Namespace`). So a template links
an asset as `/static/core/<path-relative-to-static>`, e.g.:

```html
<link rel="stylesheet" href="/static/core/css/bulma.min.css">
<link rel="stylesheet" href="/static/core/css/theme.css">
<script src="/static/core/js/navbar.js" defer></script>
```

`themetest`'s stylesheet check reads `base.p2`, finds every `/static/core/…` link,
and fails if the asset tree does not contain that file. **Add an asset → put it under
`static/` and link the matching `/static/core/` path, or the suite fails** (a page
that renders fine but is unstyled is the failure this catches).

## Theme contract (the load-bearing part)

A theme **replaces** the framework's own pages by *substitution*, not overlay. `@core/`
is an absolute namespace that resolves in exactly one root with **no fallback**, so the
theme's `base.p2` becomes the one the framework's error pages `{% extends %}`. A theme
is all-or-nothing: it is refused at startup unless every required template is present
**and compiles**.

- **`Required` is not a hand-maintained list.** `register.go` sets
  `Required = webapp.RequiredTemplates()`, read from the framework at runtime. Do not
  hardcode the template names — that copy would lag the framework quietly. The current
  set is `@core/base.p2`, `@core/login.p2`, `@core/fragments/nav.p2`,
  `@core/errors/{notfound,unauthorized,internal}.p2`.
- Required entries are `@core/`-qualified. A bare name resolves to whichever mounted
  root came first, so the check would pass against an *application's* copy while the
  theme's is missing. This is enforced by `TestRequiredMatchesFrameworkContract`.
- `p2/errors/base.p2` ships but is **not** required: it is the shared error skeleton
  extended by the three error children, and lives in `themetest.Excused`.
- **Every `.p2` you add must be in `Required` or `themetest.Excused`**, or
  `themetest`'s "nothing-unverified" check fails. Adding a template means a framework
  change, not just a file drop here.

### Blocks must be declared, not just filled

pongo2 silently drops a `{% block %}` a child fills when the parent never declared it
— no parse error, no render error, identical-looking output. `themetest` checks the
parent side. The required block set (framework's `themeblocks.Required`):

| Template | Must declare blocks |
| --- | --- |
| `base.p2` | `title`, `head`, `bodyclass`, `header`, `content`, `footer` |
| `login.p2` | `title`, `content` |
| `errors/base.p2` | `title`, `content`, `detail` |

All three error children fill `detail`; `login.p2` opts out of the chrome by
overriding `header`/`footer` to empty — which is why those elements live *inside*
their blocks in `base.p2`, not around them.

## Data contract the framework hands the templates

These come from the framework, not from anything in this repo — match the names, and
do not invent permission logic:

- Layout: `siteTitle`, `buildVersion`.
- Nav (`fragments/nav.p2`): `navElements` (each item has `.Label .Href .Base .Icon
  .Children`), `pageBase` (active item = one whose `.Base` matches it), `isAuthenticated`,
  `user.Identity`, `signInURL`, `signOutURL`. **Items arrive pre-filtered by the
  authorization layer** — a template that re-implements "can I see this" is a second,
  disagreeable answer to the same question.
- `csrfInput` is rendered **raw** (`{{ csrfInput }}`), already an HTML input.
- Login: `loginError` (one indistinguishable message for every failure — enumeration
  defense), `action`, `username` (echoed back), `next`. Password is never repopulated.
- Errors: `status`, `statusText`, `requestId`, `errorDetail` (rendered only when the
  framework says disclosure is allowed), `requiredPerm`.
- `nav.p2` renders only the **inside** of an already-opened navbar — never emit its
  own `<nav>` (the parent controls the Bulma container; nesting one navigation in
  another is the default theme's bug, not a pattern to copy).

## Test-enforced rules (the "why" behind the suite)

`bulma_test.go` runs `themetest.Check(...)` (the framework's published contract suite,
same as startup) plus repo-specific guards. Know these before editing, because they
fail in ways that look unrelated:

- **No inline script/handlers in `base.p2`**: no `onclick=`, `onfocus=`, `onload=`,
  `<script>`. The theme must not require inline JS, or an app setting CSP
  `script-src 'self'` loses its navigation. That is why both toggles live in
  `navbar.js`/`theme.js`, not in templates. (`<script src=…>` is fine; only inline
  `<script>` is forbidden.)
- **No `--wa-*` custom properties anywhere in `p2/`.** Those are the *default* theme's
  private variables; the page looks right only in the theme it was written against.
  Style with Bulma's own vocabulary; define anything else in `theme.css`.
- **`navbar.js` must reference `wa-nav-toggle`** — the id `base.p2` renders. The two
  are checked for drift.
- **The light/dark toggle is three files kept in sync** (`TestThemeToggleIsWired`):
  `theme.js` references both `wa-theme-toggle` and `data-theme`; `base.p2` loads
  `js/theme.js`; `fragments/nav.p2` renders `id="wa-theme-toggle"`. The navbar follows
  the scheme (plain `navbar`, not `is-dark`); the scheme itself is Bulma's, driven by
  `data-theme` on `<html>`, which the vendored bundle already supports.
- **`theme.css` must keep these exact fragments** (each is a fix the bundle does not
  provide; the test message explains each): `min-height: 100vh`, `flex: 1 0 auto`,
  `body.wa-login-page #main > .container`, `flex-grow: 0`, and
  `color: var(--bulma-navbar-item-color)`. `flex-grow: 0` must precede the centering
  margins because Bulma gives `.container` `flex-grow: 1`. The burger color rule exists
  because Bulma paints the burger in the navbar's own background color, giving a
  contrast ratio of one.
- **Bundle integrity**: `TestVendoredBundle` size-bands the css and checks for markers
  (`prefers-color-scheme`, `.navbar-burger`, `.button.is-primary`, `--bulma-scheme-main`);
  `TestLockMatchesBundle` checks the version string appears in the css; `TestLicenseVendored`
  requires `MIT`; `TestCommittedBundleMatchesLockDigests` recomputes every `sha256:` in
  `bulma.lock` (asserting exactly **2** per-file digests — bundle + licence).

## Advancing the vendored Bulma release

Only touch `bulma.lock` and `static/css/bulma.{min.css,LICENSE}` through this flow. The
lock is the single source of truth for the version (there is no version constant to
drift). The bundle is committed *because* a library consumer cannot run `go generate`
over a dependency, so a mismatch is a test failure, not something shipped.

1. Edit `bulma.lock`: set `version` and the digests. `go run ./internal/fetchbulma
   -version X.Y.Z -digest` prints what to paste (report-only; writes nothing).
2. `make generate` — downloads the locked release, **refuses** anything whose tarball
   or per-file digest the lock does not record, then writes the bundle.
3. `make test` — including the contract suite, which is what says the templates still
   satisfy what the framework's Go code expects.

`fetchbulma` flags: `-check` (offline, refuses to check a non-locked version),
`-digest` (print digests, no write), `-list` (package contents), `-version X` (a
version other than locked is always report-only, never a write). It only ever copies
`package/css/bulma.min.css` and `package/LICENSE` from the npm tarball (see `fileMap`);
the SCSS source is intentionally not vendored so no Sass compiler is needed at runtime.

## Don'ts

- Don't hand-edit `static/css/bulma.min.css` or `static/css/LICENSE`; advance the
  lock and regenerate. (`theme.css` and `navbar.js` *are* hand-written — edit freely.)
- Don't hardcode the `Required` template list; read it from `webapp.RequiredTemplates()`.
- Don't pull the `webapp` import outside `register.go`.
- Don't add `onclick`/inline script to templates, or `--wa-*` properties.
- Don't re-implement nav authorization in templates.
- Don't push to a remote or cut a tag unless explicitly asked.
