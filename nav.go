package main

import (
	"encoding/base64"
	"log"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

type nav struct {
	app.Compo
	sh            *shell.Shell
	loggedIn      bool
	termsAccepted bool
	userID        string
}

func newNav() *nav {
	return &nav{}
}

func (n *nav) OnMount(ctx app.Context) {
	ctx.GetState("loggedIn", &n.loggedIn)
	if n.loggedIn {
		ctx.GetState("userID", &n.userID)
		sh := shell.NewShell("localhost:5001")
		n.sh = sh
	}

	ctx.GetState("termsAccepted", &n.termsAccepted)
}

func (n *nav) doOverlay(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("content").Get("classList").Call("toggle", "overlay")
}

func (n *nav) acceptTerms(ctx app.Context, e app.Event) {
	n.termsAccepted = true
	ctx.SetState("termsAccepted", true).Persist()
	app.Window().GetElementByID("main-menu").Call("click")
}

func (n *nav) deleteAccount(ctx app.Context, e app.Event) {
	e.PreventDefault()
	n.deleteUser()
	n.deleteBalance()
	ctx.DelState("termsAccepted")
	ctx.Reload()

}

func (n *nav) deleteUser() {
	userId := base64.StdEncoding.EncodeToString([]byte(n.userID))
	err := n.sh.OrbitDocsDelete(dbUser, string(userId))
	if err != nil {
		log.Fatal(err)
	}
}

func (n *nav) deleteBalance() {
	err := n.sh.OrbitDocsDelete(dbUserBalance, n.userID)
	if err != nil {
		log.Fatal(err)
	}
}

func (n *nav) Render() app.UI {
	return app.Nav().Body(
		app.Div().Class("navbar").Body(
			app.Div().Class("container nav-container").Body(
				app.Input().ID("main-menu").Class("checkbox").Type("checkbox").Name("Main Menu").OnClick(n.doOverlay),
				app.Div().Class("hamburger-lines").Body(
					app.Span().Class("line line1"),
					app.Span().Class("line line2"),
					app.Span().Class("line line3"),
				),
				app.If(!n.loggedIn, func() app.UI {
					return app.Div().Class("menu-items").Body(
						app.Li().Body(
							app.A().Href("/terms").Target("_blank").Text("Terms of Use"),
						),
						app.Li().Body(
							app.A().Href("/privacy").Target("_blank").Text("Privacy"),
						),
						app.Li().Body(
							app.A().Href("/cookie").Target("_blank").Text("Cookie"),
						),
						app.If(!n.termsAccepted, func() app.UI {
							return app.Div().Class("menu-btn").Body(
								app.Button().ID("accept-terms").Class("submit").Type("submit").Text("Accept Terms").OnClick(n.acceptTerms),
							)
						}),
					)
				}).Else(func() app.UI {
					return app.Div().Class("menu-items").Body(
						app.Li().Body(
							app.A().Href("/wallet").Text("Wallet"),
						),
						app.Li().Body(
							app.A().Href("/payment").Text("Payment"),
						),
						app.Li().Body(
							app.A().Href("/terms").Text("Terms of Use"),
						),
						app.Li().Body(
							app.A().Href("/privacy").Text("Privacy"),
						),
						app.Li().Body(
							app.A().Href("/cookie").Text("Cookie"),
						),
						app.Li().Body(
							app.A().Text("Delete Account").OnClick(n.deleteAccount),
						),
					)
				}),
			),
		),
	)
}
