package main

import (
	"log"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// wallet is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type wallet struct {
	app.Compo
	loggedIn bool
	user     *User
	userId   string
}

func (w *wallet) OnMount(ctx app.Context) {
	ctx.GetState("loggedIn", &w.loggedIn)
	if !w.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("user", &w.user)

	log.Println(string(w.user.ID))
	w.userId = string(w.user.ID)
}

// The Render method is where the component appearance is defined. Here, a
// wallet is displayed.
func (w *wallet) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				app.Div().Class("header-summary").Body(
					app.Span().ID("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Balance"),
					),
					app.Div().Class("summary-balance").Body(
						app.Span().Text("GUBI 293.00"),
					),
				),
			),
			app.Div().Class("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Monthly Recurring"),
							app.Span().Text("3000 GUBI"),
						),
					),
					app.Div().Class("lower-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("My Payment ID"),
							app.Span().Text(w.userId),
						),
					),
				),
				app.Div().Class("transactions").Body(
					app.Span().Class("t-desc").Text("Recent Transactions"),
					app.Div().Class("transaction").Body(
						app.Div().Class("t-details").Body(
							app.Div().Class("t-title").Body(
								app.Span().Text("99 designs"),
							),
							app.Div().Class("t-time").Body(
								app.Span().Text("03.45 PM"),
							),
						),
						app.Div().Class("t-amount").Body(
							app.Span().Text("-100 GUBI"),
						),
					),
					app.Div().Class("transaction").Body(
						app.Div().Class("t-details").Body(
							app.Div().Class("t-title").Body(
								app.Span().Text("99 designs"),
							),
							app.Div().Class("t-time").Body(
								app.Span().Text("03.45 PM"),
							),
						),
						app.Div().Class("t-amount").Body(
							app.Span().Text("-100 GUBI"),
						),
					),
					app.Div().Class("transaction").Body(
						app.Div().Class("t-details").Body(
							app.Div().Class("t-title").Body(
								app.Span().Text("99 designs"),
							),
							app.Div().Class("t-time").Body(
								app.Span().Text("03.45 PM"),
							),
						),
						app.Div().Class("t-amount").Body(
							app.Span().Text("-100 GUBI"),
						),
					),
				),
			),
			app.Div().Class("drawer").Body(
				app.Div().Class("menu-btn").Body(
					app.Span().Text("Pay"),
				),
			),
		),
	)
}
