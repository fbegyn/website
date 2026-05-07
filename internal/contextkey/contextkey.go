// Package contextkey provides a typed key for stuffing values into a
// context.Context. Using a dedicated string subtype avoids collisions
// with values keyed under bare strings by other packages.
package contextkey

// Key is the typed wrapper; use as `ctx.Value(contextkey.Key("foo"))`.
type Key string
