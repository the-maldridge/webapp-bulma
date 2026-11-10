/*
Package bulma is an alternative theme for the webapp framework: the base layout,
login form, navigation fragment, and error pages written in Bulma's markup
instead of the default theme's.

It is a rewrite rather than a wrapper. Bulma's components are a contract about
structure, not about class names sprinkled on markup chosen elsewhere: a
`.field` expects a `.label` and a `.control` beneath it, and the default theme's
`<label class="field">` is not that, however much of Bulma is loaded. So this
package owns its templates outright and vendors only the stylesheet.

The stylesheet is committed. It used to be fetched by `go generate`, and that made
this theme unusable from any application that imported it as a library, because
go generate refuses to run over a package belonging to a dependency, which left
the bundle missing from the consumer's build and every page it rendered unstyled.
Committing it is what allows the theme to live here, in its own repository,
instead of only inside the framework's. The bundle's licence travels with it for
the same reason the templates are distinguished from it: the templates are this
module's own work and the stylesheet is not.

A release is advanced by editing bulma.lock and running `go generate`, which
fetches that release and refuses to write anything whose digest the lock does not
record. `fetchbulma -check` answers, without the network, whether what is on disk
is what the lock says, and the tests assert the digests directly so that a
committed bundle which does not match its own lock is a test failure rather than
something shipped.

Importing this package registers the theme, after which WithTheme("bulma") or
<PREFIX>_THEME=bulma selects it.
*/
package bulma

import (
	"embed"
	"io/fs"
)

//go:generate go run ./internal/fetchbulma

//go:embed p2
var embedded embed.FS

//go:embed static
var embeddedStatic embed.FS

// FS returns the theme's templates with the p2 directory as its root.
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "p2")
	if err != nil {
		// Unreachable: the embed directive above references the directory
		// literally, so a build in which "p2" is absent does not compile.
		panic("bulma: embedded template directory is missing: " + err.Error())
	}
	return sub
}

// Static returns the vendored Bulma bundle, served by the core under its static
// root so the base layout can link it the way any theme's base layout does.
func Static() fs.FS {
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		panic("bulma: embedded static directory is missing: " + err.Error())
	}
	return sub
}
