package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type nav struct {
	app.Compo
}

func newNav() *nav {
	return &nav{}
}

func (n *nav) doOverlay(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("content").Get("classList").Call("toggle", "overlay")
}

func (n *nav) Render() app.UI {
	return app.Nav().Body(
		app.Div().Class("navbar").Body(
			app.Div().Class("container nav-container").Body(
				app.Input().ID("").Class("checkbox").Type("checkbox").Name("").OnClick(n.doOverlay),
				app.Div().Class("hamburger-lines").Body(
					app.Span().Class("line line1"),
					app.Span().Class("line line2"),
					app.Span().Class("line line3"),
				),
				app.Div().Class("menu-items").Body(
					app.Li().Body(
						app.A().Href("/wallet").Text("Wallet"),
					),
					app.Li().Body(
						app.A().Href("/payment").Text("Payment"),
					),
					app.Li().Body(
						app.A().Href("/delete-account").Text("Delete Account"),
					),
				),
			),
		),
	)
}
