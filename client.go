package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

// client is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type client struct {
	app.Compo
	sh            *shell.Shell
	loggedIn      bool
	businessName  string
	userBalance   UserBalance
	plan          Plan
	subscriptions []Subscription
	totalIncome   int
}

func (c *client) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	c.sh = sh

	ctx.GetState("loggedIn", &c.loggedIn)
	if !c.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("businessName", &c.businessName)

	ctx.GetState("balance", &c.userBalance)

	ctx.GetState("plan", &c.plan)

	c.getSubscriptions(ctx)
}

func (c *client) getSubscriptions(ctx app.Context) {
	ctx.Async(func() {
		subs, err := c.sh.OrbitDocsQuery(dbSubscription, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		subscriptions := []Subscription{}
		var totalIncome int

		if len(subs) != 0 {
			err = json.Unmarshal(subs, &subscriptions) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}

			for _, sub := range subscriptions {
				totalIncome += sub.Price
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.subscriptions = subscriptions
			c.totalIncome = totalIncome
		})
	})
}

// The Render method is where the component appearance is defined. Here, a
// client is displayed.
func (c *client) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Balance"),
					),
					app.Div().Class("summary-balance").Body(
						app.Span().Text(strconv.Itoa(c.userBalance.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Business Name"),
							app.Span().Class("span-body").Text(c.businessName),
						),
					),
					app.Div().Class("lower-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Recurring Income"),
							app.Span().Class("span-body").Text(strconv.Itoa(c.totalIncome/100)+" GUBI"),
						),
					),
				),
				app.Div().Class("subscriptions").Body(
					app.Span().Class("s-desc").Text("Recent Subscriptions"),
					app.If(c.plan == Plan{} || len(c.subscriptions) == 0, func() app.UI {
						return app.Div().Class("subscription").Body(
							app.Span().Class("empty").Text("No subscriptions yet"),
						).Style("pointer-events", "none")
					}),
					app.Range(c.subscriptions).Slice(func(i int) app.UI {
						return app.Div().Class("subscription c-sub").Body(
							app.Div().Class("s-details").Body(
								app.Div().Class("c-title").Body(
									app.Span().Text("User ID: "+c.subscriptions[i].UserID),
								),
								app.Div().Class("s-time").Body(
									app.Span().Text(c.subscriptions[i].StartDate.Format("2006-01-02 15:04")),
									app.Span().Text(c.subscriptions[i].EndDate.Format("2006-01-02 15:04")),
								),
							),
							app.Div().Class("s-price").Body(
								app.Span().Text(strconv.Itoa(c.plan.Price/100)+" GUBI"),
							),
						)
					}),
				),
			),
		),
	)
}
