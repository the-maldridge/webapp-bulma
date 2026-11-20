package bulma_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/the-maldridge/webapp"
	"github.com/the-maldridge/webapp-bulma"
	"github.com/the-maldridge/webapp/themetest"
)

// bundlePath is the vendored stylesheet this theme's base layout links, below
// the package's own static root.
const bundlePath = "static/css/bulma.min.css"

// licensePath is the bundle's own licence, which travels with it.
const licensePath = "static/css/LICENSE"

// navScript is the one script this theme writes itself, and unlike the bundle it
// is committed.
const navScript = "static/js/navbar.js"

// themeCSS is the theme's own stylesheet: page structure the bundle's helper
// classes do not express, linked by base.p2 beside the bundle.
const themeCSS = "static/css/theme.css"

// lockPath is the committed record of which release the bundle is.
const lockPath = "bulma.lock"

// minBundleBytes and maxBundleBytes bracket a real Bulma release. The bounds are
// loose on purpose: they exist to catch a truncated download, an HTML error page
// saved as a stylesheet, and a bundle that grew by an order of magnitude because
// something other than the minified CSS landed here.
const (
	minBundleBytes = 100_000
	maxBundleBytes = 2_000_000
)

// TestThemeContract runs the rules every theme has to satisfy against this one.
//
// It is the same checker the core runs over registered themes, run here so that a
// theme with an incomplete required set fails `go test .` in this repository rather
// than only someone's application startup.
//
// Nothing in this suite skips. The bundle is committed, so every assertion runs on
// every invocation. That is a deliberate change from when the bundle was generated:
// a check that could be made to disappear by leaving a file out reports green for
// the parts of the theme that were not built at all.
func TestThemeContract(t *testing.T) {
	themetest.Check(t, "bulma", bulma.FS(), bulma.Static(), bulma.Required)
}

// TestRequiredMatchesFrameworkContract asserts this theme supplies exactly the
// templates the framework says it renders itself.
//
// The required set is fixed by what the framework renders, so a theme adding to or
// dropping from it is either papering over a framework change or discovering a page
// the framework renders that this theme never heard of. Both are worth a failing
// test rather than a quiet divergence. The contract is read from the framework
// rather than from a sibling theme, because the framework is what makes it.
func TestRequiredMatchesFrameworkContract(t *testing.T) {
	want := strings.Join(webapp.RequiredTemplates(), "\n")
	got := strings.Join(bulma.Required, "\n")
	if want != got {
		t.Errorf("Required differs from what the framework says it renders:\n"+
			"framework:\n%s\nbulma:\n%s", want, got)
	}
}

// TestVendoredBundle checks the committed stylesheet is what this theme thinks it
// is linking.
func TestVendoredBundle(t *testing.T) {
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) < minBundleBytes || len(data) > maxBundleBytes {
		t.Errorf("%s is %d bytes, outside the %d..%d band for a Bulma release; "+
			"a truncated download or an error page saved as a stylesheet looks exactly like this",
			bundlePath, len(data), minBundleBytes, maxBundleBytes)
	}

	// The dark scheme is the part of the bundle most likely to be missing from a
	// partial build, and it is a rule a minifier would never drop on its own.
	if !strings.Contains(string(data), "prefers-color-scheme") {
		t.Error("bundle contains no prefers-color-scheme rule, so it is not the CSS build it claims to be")
	}
	for _, marker := range []string{"--bulma-scheme-main", ".navbar-burger", ".button.is-primary"} {
		if !strings.Contains(string(data), marker) {
			t.Errorf("bundle does not define %s, which this theme's templates depend on", marker)
		}
	}
}

// TestLockMatchesBundle asserts the committed lock file describes the bundle on
// disk, so that the bundle and the record of what it should be cannot drift apart
// while tests still pass.
//
// This is the loose half of the check: a version string inside a file proves
// provenance and nothing else. TestCommittedBundleMatchesLockDigests is the strict
// half, and the two together are the reason the bundle can be committed at all.
func TestLockMatchesBundle(t *testing.T) {
	version := lockValue(t, "version")
	if version == "" {
		t.Fatalf("%s records no version", lockPath)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version) {
		t.Errorf("%s version %q is not a release version", lockPath, version)
	}

	// The bundle is the release's own file, so the version has to appear in it:
	// Bulma stamps a header comment, and a bundle built from a different release
	// is the failure this check is for.
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), version) {
		t.Errorf("%s pins %s but %s does not contain that version string", lockPath, version, bundlePath)
	}
}

// TestLicenseVendored asserts the bundle's licence travels with it.
//
// The templates are this repository's own work, but the stylesheet below them is
// not, and a theme that vendors CSS without the text has distributed it
// unlawfully in every application that imports it.
func TestLicenseVendored(t *testing.T) {
	data, err := os.ReadFile(licensePath)
	if err != nil {
		t.Fatalf("%s is missing; the vendored bundle cannot be redistributed without it: %v", licensePath, err)
	}
	if !strings.Contains(string(data), "MIT") {
		t.Errorf("%s does not read like the licence the bundle was released under", licensePath)
	}
}

// TestNoFrameworkCustomProperties asserts this theme does not reach for the
// default theme's stylesheet variables.
//
// Nothing in a theme may depend on another theme's custom properties, because
// they are that theme's private business. The mistake is easy to make and hard
// to see, since the page looks right in whichever theme it was written against.
func TestNoFrameworkCustomProperties(t *testing.T) {
	seen := map[string]string{}
	err := fs.WalkDir(bulma.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".p2" {
			return nil
		}
		src, err := fs.ReadFile(bulma.FS(), p)
		if err != nil {
			return err
		}
		if i := strings.Index(string(src), "--wa-"); i >= 0 {
			seen[p] = string(src)[i:min(len(string(src)), i+40)]
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	for p, ctx := range seen {
		t.Errorf("%s references the default theme's custom properties (%s); a theme styles with its "+
			"own framework's vocabulary and defines whatever else it needs itself", p, ctx)
	}
}

// TestRegistrationRuns asserts importing this package registers the theme under
// the name configuration uses.
//
// The check is that WithTheme accepts the name, because that is the whole of what
// an application does with it, and it is the failure mode a missing register.go
// produces: templates that exist, assets that exist, and a configuration variable
// that matches nothing.
func TestRegistrationRuns(t *testing.T) {
	var cfg webapp.Config
	if err := webapp.WithTheme("bulma")(&cfg); err != nil {
		t.Fatalf("WithTheme(\"bulma\"): %v; the theme's init must register it", err)
	}
}

// TestCommittedBundleMatchesLockDigests verifies the committed bundle against the
// digests in the lock file, without the network.
//
// Committing the stylesheet is what lets this theme be used from another module, so
// the bundle a consumer gets is whatever was committed rather than something the
// build just fetched and verified. That makes this the only check standing between a
// silently edited, truncated, or mis-saved file and every application that imports
// this theme. The version substring assertion in TestLockMatchesBundle cannot catch
// any of it: a 1.0.4 header survives an arbitrary amount of damage to the bytes
// beneath it.
func TestCommittedBundleMatchesLockDigests(t *testing.T) {
	src, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", lockPath, err)
	}

	var checked int
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		key, want, ok := strings.Cut(line, "=")
		if !ok || !strings.HasPrefix(key, "sha256:") {
			continue
		}
		name := strings.TrimPrefix(key, "sha256:")

		data, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("%s is listed in %s but is not in the module: %v", name, lockPath, err)
			continue
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s has digest %s but %s says %s; the committed bundle is not the artifact the "+
				"lock records, so whatever was reviewed upstream is not what consumers embed",
				name, got, lockPath, want)
		}
		checked++
	}

	// A lock file that stopped naming any per-file digest would make the loop above
	// vacuously true, which is the failure this guards and so is itself a failure.
	if checked != 2 {
		t.Errorf("%s records %d per-file digests, want 2 (the bundle and its licence)",
			lockPath, checked)
	}
}

// TestBundledScriptIsServed asserts the theme's own JavaScript is in the asset
// tree the core serves, which is what makes the link in base.p2 resolve.
//
// It is checked here because the failure is silent in every sense a test normally
// catches: a page that loads with a 200, looks correct, and has a navigation menu
// that does not open on a phone.
func TestBundledScriptIsServed(t *testing.T) {
	data, err := fs.ReadFile(bulma.Static(), "js/navbar.js")
	if err != nil {
		t.Fatalf("%s is missing from the asset tree (%v), so base.p2 links a script that answers 404",
			navScript, err)
	}
	if !strings.Contains(string(data), "wa-nav-toggle") {
		t.Errorf("%s does not reference the toggle base.p2 renders, so the two have drifted apart",
			navScript)
	}
}

// TestOwnStylesheetIsServed is the same guarantee as the script test, for the
// stylesheet: the link in base.p2 resolves, and the file still carries the
// sticky-footer rules the layout relies on. The bundle ships no standalone
// min-height helper, so these rules live here and nowhere else; losing one
// silently returns every footer to floating under the last line of a short
// page, which is the bug this stylesheet was written for.
func TestOwnStylesheetIsServed(t *testing.T) {
	data, err := fs.ReadFile(bulma.Static(), "css/theme.css")
	if err != nil {
		t.Fatalf("%s is missing from the asset tree (%v), so base.p2 links a stylesheet that answers 404",
			themeCSS, err)
	}
	rules := []struct{ fragment, why string }{
		{"min-height: 100vh", "keeps the footer at the bottom of the content or the viewport, whichever is lower"},
		{"flex: 1 0 auto", "lets the main region eat the spare height the footer rule creates"},
		{"body.wa-login-page #main > .container", "holds the sign-in box in the middle of the window"},
		{"flex-grow: 0", "cancels the bundle's own flex-grow on .container, without which the auto margins have nothing to centre"},
	}
	for _, r := range rules {
		if !strings.Contains(string(data), r.fragment) {
			t.Errorf("%s no longer carries %q, which %s", themeCSS, r.fragment, r.why)
		}
	}
}

// TestNoInlineHandlers asserts the theme ships no inline JavaScript or event
// attributes.
//
// A useful Content-Security-Policy is specific to the application, and an
// application is free to set script-src 'self'. A theme whose layout carried an
// onclick attribute would quietly break navigation for exactly the deployments that
// got their policy right, and the theme author would never see it, so inline script
// is not something this theme may require.
func TestNoInlineHandlers(t *testing.T) {
	src, err := fs.ReadFile(bulma.FS(), "base.p2")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, forbidden := range []string{" onclick=", " onfocus=", " onload=", "<script>"} {
		if strings.Contains(string(src), forbidden) {
			t.Errorf("base.p2 contains %q; the theme must not require inline script, because an "+
				"application setting Content-Security-Policy: script-src 'self' would lose its "+
				"navigation", forbidden)
		}
	}
}

// TestFetcherCheckVerifiesDigests runs the generator's offline check and then
// corrupts a bundle in a copy of the package to confirm the check fails.
//
// The corrupt case is the point. A check that only proves a file exists passes for
// a truncated download and for an error page saved under a stylesheet's name, and
// those are the two failures that produce a build which works and a page that does
// not.
func TestFetcherCheckVerifiesDigests(t *testing.T) {
	binary := buildFetcher(t)

	if out := runFetcher(t, binary, ".", "-check"); !strings.Contains(out, lockValue(t, "version")) {
		t.Errorf("-check on a matching tree did not report the locked version: %s", out)
	}

	// A scratch copy, because the corrupted file must not outlive the test. The real
	// bundle is committed, so a stray write here would be a change to the artifact
	// every consumer embeds rather than to regenerable output.
	dir := t.TempDir()
	for _, name := range []string{lockPath, bundlePath, licensePath} {
		copyInto(t, name, filepath.Join(dir, name))
	}

	data, err := os.ReadFile(filepath.Join(dir, bundlePath))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	data[len(data)/2] ^= 0xff
	if err := os.WriteFile(filepath.Join(dir, bundlePath), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if out, err := runFetcherAllowFail(t, binary, dir, "-check"); err == nil {
		t.Errorf("-check accepted a bundle whose digest does not match %s: %s", lockPath, out)
	} else if !strings.Contains(out, bundlePath) {
		t.Errorf("-check's failure does not name the file at fault: %s", out)
	}
}

// buildFetcher compiles the vendored generator so it can be run against scratch
// directories the tests control.
func buildFetcher(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "fetchbulma")
	cmd := exec.Command("go", "build", "-o", binary, "./internal/fetchbulma")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the generator: %v\n%s", err, out)
	}
	return binary
}

// runFetcher runs the generator in dir and returns its combined output, failing
// the test if it exits non-zero.
func runFetcher(t *testing.T, binary, dir string, args ...string) string {
	t.Helper()

	out, err := runFetcherAllowFail(t, binary, dir, args...)
	if err != nil {
		t.Fatalf("fetchbulma %v: %v\n%s", args, err, out)
	}
	return out
}

// runFetcherAllowFail runs the generator and reports both its output and whether
// it succeeded.
func runFetcherAllowFail(t *testing.T, binary, dir string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// copyInto copies one file into a destination, creating its directory.
func copyInto(t *testing.T, src, dst string) {
	t.Helper()

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", dst, err)
	}
}

// lockValue reads one key from the lock file.
func lockValue(t *testing.T, key string) string {
	t.Helper()

	src, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if k == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// TestLockFileSize is a smoke test that the lock file is parseable in the same
// way the generator reads it, including its byte counts where recorded.
func TestLockFileShape(t *testing.T) {
	src, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	seen := map[string]bool{}
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			t.Errorf("%s has a malformed line %q; the generator refuses to read a lock file it cannot "+
				"parse in full, and so does this", lockPath, line)
			continue
		}
		seen[k] = true
		if k == "version" {
			if _, err := strconv.Atoi(strings.Split(v, ".")[0]); err != nil {
				t.Errorf("version %q does not start with a number: %v", v, err)
			}
		}
		if strings.HasPrefix(k, "sha256:") && len(v) != 64 {
			t.Errorf("digest for %s is %d characters, not 64", k, len(v))
		}
	}
	for _, key := range []string{"version", "tarball_sha256"} {
		if !seen[key] {
			t.Errorf("%s records no %s", lockPath, key)
		}
	}
}
