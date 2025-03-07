package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type cookie struct {
	app.Compo
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (c *cookie) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().ID("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Authentication"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item card-terms").Body(
							app.Span().Class("span-header").Text("Cookie Policy"),
							app.Span().Class("span-docs").Text("1. Cyber-gubi does not use cookies."),
							app.Span().Class("span-docs").Text("1. Cyber-gubi does not track you."),
							app.Span().Class("span-docs").Text("1. Cyber-gubi does not share data with 3rd parties."),
						),
					),
				),
			),
		),
	)
}
