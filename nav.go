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
	isBusiness    bool
	businessName  string
	vat           string
	entity        string
	userID        string
	plan          Plan
}

func newNav() *nav {
	return &nav{}
}

func (n *nav) OnMount(ctx app.Context) {
	ctx.GetState("loggedIn", &n.loggedIn)
	if n.loggedIn {
		ctx.GetState("userID", &n.userID)
		ctx.GetState("isBusiness", &n.isBusiness)
		sh := shell.NewShell("localhost:5001")
		n.sh = sh
	}

	ctx.ObserveState("termsAccepted", &n.termsAccepted)
	ctx.ObserveState("entity", &n.entity)
	// ctx.ObserveState("businessName", &n.businessName)
	// ctx.ObserveState("vat", &n.vat)
	ctx.ObserveState("plan", &n.plan)
}

func (n *nav) doOverlay(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("content").Get("classList").Call("toggle", "overlay")
}

func (n *nav) acceptTermsIndividual(ctx app.Context, e app.Event) {
	e.PreventDefault()
	ctx.SetState("termsAccepted", true)
	app.Window().GetElementByID("main-menu").Call("click")
}

func (n *nav) acceptTermsBusiness(ctx app.Context, e app.Event) {
	e.PreventDefault()
	ctx.SetState("termsAccepted", true)
}

func (n *nav) deleteAccount(ctx app.Context, e app.Event) {
	e.PreventDefault()
	n.deleteUser()
	n.deleteBalance()
	// delete subscriptions
	if n.isBusiness {
		n.deletePlan()
		// delete clients
		// delete suppliers
	}
	ctx.DelState("termsAccepted")
	ctx.Reload()

}

func (n *nav) deletePlan() {
	err := n.sh.OrbitDocsDelete(dbPlan, n.plan.ID)
	if err != nil {
		log.Fatal(err)
	}
}

func (n *nav) registerIndividual(ctx app.Context, e app.Event) {
	e.PreventDefault()
	ctx.SetState("entity", "individual")
}

func (n *nav) registerBusiness(ctx app.Context, e app.Event) {
	e.PreventDefault()
	ctx.SetState("entity", "business")
}

func (n *nav) submitVAT(ctx app.Context, e app.Event) {
	validVAT := app.Window().GetElementByID("vat-number").Call("reportValidity").Bool()
	validBusinessName := app.Window().GetElementByID("business-name").Call("reportValidity").Bool()
	if validVAT && validBusinessName {
		businessName := app.Window().GetElementByID("business-name").Get("value").String()
		associateName := app.Window().GetElementByID("associate-name").Get("value").String()
		vat := app.Window().GetElementByID("vat-number").Get("value").String()
		ctx.SetState("vat", vat)
		ctx.SetState("businessName", businessName)
		ctx.SetState("associateName", associateName)
		app.Window().GetElementByID("main-menu").Call("click")
	}
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
				app.If(n.loggedIn, func() app.UI {
					return app.Input().ID("main-menu").Class("checkbox").Type("checkbox").Name("Main Menu").OnClick(n.doOverlay)
				}).Else(func() app.UI {
					return app.Input().ID("main-menu").Class("checkbox").Type("checkbox").Name("Main Menu").OnClick(n.doOverlay).Style("pointer-events", "none")
				}),
				app.If(n.loggedIn, func() app.UI {
					return app.Div().Class("hamburger-lines").Body(
						app.Span().Class("line line1"),
						app.Span().Class("line line2"),
						app.Span().Class("line line3"),
					)
				}).Else(func() app.UI {
					return app.Div().Class("hamburger-lines").Body(
						app.Span().Class("line line1"),
						app.Span().Class("line line2"),
						app.Span().Class("line line3"),
					).Style("display", "none")
				}),

				app.If(!n.loggedIn, func() app.UI {
					return app.If(!n.termsAccepted, func() app.UI {
						return app.If(n.entity == "individual", func() app.UI {
							return app.Div().Class("menu-items").Body(
								app.Div().Class("header-summary").Body(
									app.Span().Class("logo").Text("cyber-gubi"),
									app.Div().Class("summary-text").Body(
										app.Span().Text("Individual"),
									),
								),
								app.Li().Body(
									app.A().Href("/terms").Target("_blank").Text("Terms of Use"),
								),
								app.Li().Body(
									app.A().Href("/privacy").Target("_blank").Text("Privacy"),
								),
								app.Li().Body(
									app.A().Href("/cookie").Target("_blank").Text("Cookie"),
								),
								app.Div().Class("menu-btn").Body(
									app.Button().ID("accept-terms").Class("submit").Type("submit").Text("Accept Terms").OnClick(n.acceptTermsIndividual),
								),
							)
						}).ElseIf(n.entity == "business", func() app.UI {
							return app.Div().Class("menu-items").Body(
								app.Div().Class("header-summary").Body(
									app.Span().Class("logo").Text("cyber-gubi"),
									app.Div().Class("summary-text").Body(
										app.Span().Text("Business"),
									),
								),
								app.Li().Body(
									app.A().Href("/terms-business").Target("_blank").Text("Terms of Use"),
								),
								app.Li().Body(
									app.A().Href("/privacy-business").Target("_blank").Text("Privacy"),
								),
								app.Li().Body(
									app.A().Href("/cookie-business").Target("_blank").Text("Cookie"),
								),
								app.Div().Class("menu-btn").Body(
									app.Button().ID("accept-terms-business").Class("submit").Type("submit").Text("Accept Terms").OnClick(n.acceptTermsBusiness),
								),
							)
						}).Else(func() app.UI {
							return app.Div().Class("menu-items").Body(
								app.Div().Class("header-summary").Body(
									app.Span().Class("logo").Text("cyber-gubi"),
									app.Div().Class("summary-text").Body(
										app.Span().Text("Menu"),
									),
								),
								app.Li().Body(
									app.A().Text("For Individuals").OnClick(n.registerIndividual),
								),
								app.Li().Body(
									app.Div().Class("tooltip").DataSet("direction", "bottom").Body(
										app.Div().Class("tooltip__initiator").Body(
											app.A().Text("For Businesses").OnClick(n.registerBusiness),
										),
										app.Div().Class("tooltip__item").Text("Coming soon! Join the waitlist"),
									),
								),
							)
						})
					}).Else(func() app.UI {
						return app.If(n.entity == "business", func() app.UI {
							return app.Div().Class("menu-items").Body(
								app.Div().Class("header-summary").Body(
									app.Span().Class("logo").Text("cyber-gubi"),
									app.Div().Class("summary-text").Body(
										app.Span().Text("Business"),
									),
								),
								app.Label().Class("menu-label").For("business-name").Text("Business Name:"),
								app.Input().ID("business-name").Class("input-register").Type("text").Placeholder("Enter business name").MaxLength(22).Required(true),
								app.Label().Class("menu-label").For("associate-name").Text("Associate Name:"),
								app.Input().ID("associate-name").Class("input-register").Type("text").Placeholder("Enter associate name").MaxLength(22).Required(true),
								app.Label().Class("menu-label").For("vat-number").Text("VAT Number:"),
								app.Input().ID("vat-number").Class("input-register").Type("text").Placeholder("Enter VAT number").Required(true),
								app.Div().Class("menu-btn").Body(
									app.Button().ID("submit-vat").Class("submit").Text("Submit VAT").OnClick(n.submitVAT)),
							)
						})
					})
				}).Else(func() app.UI {
					return app.If(!n.isBusiness, func() app.UI {
						return app.Div().Class("menu-items").Body(
							app.Div().Class("header-summary").Body(
								app.Span().Class("logo").Text("cyber-gubi"),
								app.Div().Class("summary-text").Body(
									app.Span().Text("Individual"),
								),
							),
							app.Li().Body(
								app.A().Href("/wallet").Text("Wallet"),
							),
							app.Li().Body(
								app.A().Href("/payment").Text("Payment"),
							),
							app.Li().Body(
								app.A().Href("/subscriptions").Text("Subscriptions"),
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
					}).Else(func() app.UI {
						return app.Div().Class("menu-items").Body(
							app.Div().Class("header-summary").Body(
								app.Span().Class("logo").Text("cyber-gubi"),
								app.Div().Class("summary-text").Body(
									app.Span().Text("Business"),
								),
							),
							app.Li().Body(
								app.A().Href("/wallet").Text("Wallet"),
							),
							app.Li().Body(
								app.A().Href("/payment").Text("Payment"),
							),
							app.If(n.plan == Plan{}, func() app.UI {
								return app.Li().Body(
									app.A().Href("/plan").Text("Create Plan"),
								)
							}).Else(func() app.UI {
								return app.Li().Body(
									app.A().Href("/plan").Text("Edit Plan"),
								)
							}),
							app.Li().Body(
								app.A().Href("/associates").Text("Associates"),
							),
							app.Li().Body(
								app.A().Href("/clients").Text("Clients"),
							),
							app.Li().Body(
								app.A().Href("/suppliers").Text("Suppliers"),
							),
							app.Li().Body(
								app.A().Href("/terms-business").Text("Terms of Use"),
							),
							app.Li().Body(
								app.A().Href("/privacy-business").Text("Privacy"),
							),
							app.Li().Body(
								app.A().Href("/cookie-business").Text("Cookie"),
							),
							app.Li().Body(
								app.A().Text("Delete Account").OnClick(n.deleteAccount),
							),
						)
					})
				}),
			),
		),
	)
}
