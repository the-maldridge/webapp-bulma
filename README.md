# webapp-bulma

The Bulma theme for [`github.com/the-maldridge/webapp`](https://github.com/the-maldridge/webapp),
as a separate module. It is the worked example of a theme that lives outside the
framework: templates, a vendored Bulma bundle, and a `register.go`, and nothing
else. The framework never imports it; an application imports it.

## Use it

```go
import (
	"github.com/the-maldridge/webapp"
	_ "github.com/the-maldridge/webapp-bulma"
)

app, err := webapp.New(
	webapp.WithTheme("bulma"),
	// ...
)
```

Blank-importing the package registers the theme under the name `bulma` during
`init`. `WEBAPP_THEME=bulma` (whatever the application's environment prefix is)
then selects it at boot, without a rebuild.

The theme owns the markup of every page the framework renders: the base layout,
the login page, the error pages. Application templates that extend
`@core/base.p2` are untouched by the switch, which is the point.

## What is in here

| Path | The thing it is |
|---|---|
| `p2/` | The theme's templates, embedded and mounted as `@core/…` |
| `static/css/bulma.min.css` | The vendored Bulma bundle, committed |
| `static/css/LICENSE` | Bulma's MIT licence, so it travels with the binary |
| `static/js/navbar.js` | The only script the base layout links, no inline handlers |
| `bulma.lock` | The Bulma release this vendors, with the digest of every file |
| `internal/fetchbulma` | The generator that advances the bundle, `-check` to verify offline |
| `register.go` | Registration: the one file here that imports the framework |

The bundle is committed rather than generated at build time, and that is
deliberate: a library consumer's build must not need a network fetch or a
`go:generate` step it does not know about. `make test` checks the committed
files against the digests in `bulma.lock`, so a bundle that silently rotted or
was hand-edited fails the suite.

## Development

A clone of this repository alone builds and tests, because the framework is an
ordinary published dependency with a `require` and a `go.sum` like anyone else's.

The workspace is for the one thing a version pin cannot do: developing a change
that spans both repositories. Check them out side by side and let a gitignored
`go.work` point this module at the local framework.

```
parent/
├── webapp/          # github.com/the-maldridge/webapp
└── webapp-bulma/    # this repository
```

```sh
go work init .
go work use ../webapp
```

While the workspace exists it supersedes the require, so framework edits are
picked up without waiting for a release. It stays gitignored because it names a
directory that only exists in a checkout that cloned both, and somebody who
cloned only the theme has to keep resolving the framework from its published
version; committing the workspace would turn that into "directory not found".

## Advancing the Bulma release

1. Edit `bulma.lock`: point `version` at the new release and update every
   `sha256:` digest to what the upstream assets have.
2. `go generate .` — the fetcher downloads the named release, refuses anything
   whose digests do not match the lock, and writes the bundle.
3. `make test` — including the contract suite the framework publishes
   (`webapp/themetest`), which is what says the templates still satisfy what
   the framework's Go code expects of them.

`go run ./internal/fetchbulma -check` verifies the committed bundle
against the lock without any network, which is what CI would run if CI did not
just run the tests, which do the same check.
