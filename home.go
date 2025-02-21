package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// wallet is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type home struct {
	app.Compo
}

func (h *home) OnMount(ctx app.Context) {
	ctx.Navigate("auth")
}
