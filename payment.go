package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

// payment is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type payment struct {
	app.Compo
	sh          *shell.Shell
	loggedIn    bool
	userID      string
	userBalance UserBalance
}

func (p *payment) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	p.sh = sh

	// ctx.GetState("loggedIn", &p.loggedIn)
	// if !p.loggedIn {
	// 	ctx.Navigate("/auth")
	// }

	// ctx.GetState("userID", &p.userID)
	// ctx.GetState("balance", &p.userBalance)
}

func (p *payment) updateBalance(ctx app.Context) {
	ctx.Async(func() {
		userBalance := UserBalance{
			ID:      string(p.userID),
			Balance: p.userBalance.Balance,
		}

		userBalanceJSON, err := json.Marshal(userBalance)
		if err != nil {
			log.Fatal(err)
		}

		err = p.sh.OrbitDocsPut(dbUserBalance, userBalanceJSON)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func (p *payment) showProduct(ctx app.Context, e app.Event) {
	e.PreventDefault()
	app.Window().Get("document").Call("getElementById", "service-name").Call("removeAttribute", "required")
	app.Window().Get("document").Call("getElementById", "service-amount").Call("removeAttribute", "required")
	app.Window().Get("document").Call("getElementById", "tab-service").Get("classList").Call("remove", "tab-active")
	ctx.JSSrc().Get("classList").Call("add", "tab-active")
	// Hide Service Tab and show Product Tab
	app.Window().Get("document").Call("getElementById", "product-tab").Call("setAttribute", "style", "display: block")
	app.Window().Get("document").Call("getElementById", "service-tab").Call("setAttribute", "style", "display: none")
}

func (p *payment) showService(ctx app.Context, e app.Event) {
	e.PreventDefault()
	app.Window().Get("document").Call("getElementById", "product-name").Call("removeAttribute", "required")
	app.Window().Get("document").Call("getElementById", "product-price").Call("removeAttribute", "required")
	app.Window().Get("document").Call("getElementById", "product-amount").Call("removeAttribute", "required")
	app.Window().Get("document").Call("getElementById", "tab-product").Get("classList").Call("remove", "tab-active")
	ctx.JSSrc().Get("classList").Call("add", "tab-active")
	// Hide Product Tab and show Service Tab
	app.Window().Get("document").Call("getElementById", "product-tab").Call("setAttribute", "style", "display: none; ")
	app.Window().Get("document").Call("getElementById", "service-name").Call("setAttribute", "required", true)
	app.Window().Get("document").Call("getElementById", "service-amount").Call("setAttribute", "required", true)
	app.Window().Get("document").Call("getElementById", "service-tab").Call("setAttribute", "style", "display: block")
}

func (p *payment) doPay(ctx app.Context, e app.Event) {
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (p *payment) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				app.Div().Class("header-summary").Body(
					app.Span().ID("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Balance"),
					),
					app.Div().Class("summary-balance").Body(
						app.Span().Text(strconv.Itoa(p.userBalance.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().Class("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Make a payment"),
							app.Form().ID("pay-form").Body(
								app.Label().For("recipient").Text("Recipient:"),
								app.Select().ID("recipient").Name("recipient").Body(
									app.Option().Value("123456789").Text("123456789"),
									app.Option().Value("123456789").Text("123456789"),
									app.Option().Value("123456789").Text("123456789"),
								),
								// Tab Navigation
								app.Div().
									Class("tabs").
									Body(
										app.Button().
											ID("tab-product").
											Class("tab-button").
											Class("tab-active").
											Text("Product").
											OnClick(p.showProduct),
										app.Button().
											ID("tab-service").
											Class("tab-button").
											Text("Service").
											OnClick(p.showService),
									),

								// Product Tab Content
								app.Div().
									ID("product-tab").
									Class("tab-content").
									Body(
										app.Input().ID("product-name").Type("text").Name("product-name").Placeholder("Product name").Required(true),
										app.Input().ID("product-price").Type("text").Name("product-price").Placeholder("Single price").Required(true),
										app.Input().ID("product-amount").Type("text").Name("product-amount").Placeholder("Number of products").Required(true),
									),

								// Service Tab Content
								app.Div().
									ID("service-tab").
									Class("tab-content").
									Body(
										app.Input().ID("service-name").Type("text").Name("service-name").Placeholder("Service name"),
										app.Input().ID("service-amount").Type("text").Name("service-amount").Placeholder("Number of hours"),
									).Hidden(true),
								app.Div().Class("drawer drawer-pay").Body(
									app.Div().Class("menu-btn").Body(
										app.Button().Class("submit").Type("submit").Text("Pay").OnClick(p.doPay),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}
