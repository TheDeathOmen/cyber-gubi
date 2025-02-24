package main

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

const dbIncome = "income"
const dbUserBalance = "user_balance"

// wallet is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type wallet struct {
	app.Compo
	sh          *shell.Shell
	loggedIn    bool
	userID      string
	userBalance UserBalance
	income      Income
}

type UserBalance struct {
	ID           string    `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                     // Unique identifier for the user
	Balance      int       `mapstructure:"balance" json:"balance,string" validate:"uuid_rfc4122"`      // Balance of the user in cents
	Income       int       `mapstructure:"income" json:"income,string" validate:"uuid_rfc4122"`        // Recurring income of the user in cents
	LastReceived time.Time `mapstructure:"last_received" json:"last_received" validate:"uuid_rfc4122"` // Date when basic income was last received
}

type Income struct {
	ID     string    `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`              // Unique identifier for the income
	Amount int       `mapstructure:"amount" json:"amount,string" validate:"uuid_rfc4122"` // Amount of the income in cents
	Period time.Time `mapstructure:"period" json:"period" validate:"uuid_rfc4122"`        // Period the income is valid for
}

func (w *wallet) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	w.sh = sh

	// ctx.GetState("loggedIn", &w.loggedIn)
	// if !w.loggedIn {
	// 	ctx.Navigate("/auth")
	// }

	ctx.GetState("userID", &w.userID)

	// w.updateIncome()
	// w.deleteIncome()
	// w.deleteBalances()
	// return

	w.getBalance(ctx)
}

func (w *wallet) deleteBalances() {
	err := w.sh.OrbitDocsDelete(dbUserBalance, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) getBalance(ctx app.Context) {
	ctx.Async(func() {
		b, err := w.sh.OrbitDocsQuery(dbUserBalance, "_id", w.userID)
		if err != nil {
			log.Fatal(err)
		}

		if len(b) == 0 {
			ctx.Dispatch(func(ctx app.Context) {
				w.userBalance = UserBalance{}
				w.getIncome(ctx)
			})
			return
		}

		userBalances := []UserBalance{}

		err = json.Unmarshal(b, &userBalances) // Unmarshal the byte slice directly
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.userBalance = userBalances[0]
			ctx.SetState("balance", w.userBalance)
			// check if recurring income was received for this month
			if w.userBalance.LastReceived.Year() != time.Now().Year() && w.userBalance.LastReceived.Month() != time.Now().Month() {
				w.getIncome(ctx)
			}
		})
	})
}

func (w *wallet) updateBalance(ctx app.Context) {
	ctx.Async(func() {
		userBalance := UserBalance{
			ID:           string(w.userID),
			Balance:      w.userBalance.Balance,
			Income:       w.income.Amount,
			LastReceived: w.userBalance.LastReceived,
		}

		userBalanceJSON, err := json.Marshal(userBalance)
		if err != nil {
			log.Fatal(err)
		}

		err = w.sh.OrbitDocsPut(dbUserBalance, userBalanceJSON)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func (w *wallet) updateIncome() {
	income := &Income{
		ID:     uuid.NewString(),
		Amount: 100000,
		Period: time.Now(),
	}

	incomeJSON, err := json.Marshal(income)
	if err != nil {
		log.Fatal(err)
	}

	err = w.sh.OrbitDocsPut(dbIncome, incomeJSON)
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deleteIncome() {
	err := w.sh.OrbitDocsDelete(dbIncome, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) getIncome(ctx app.Context) {
	ctx.Async(func() {
		i, err := w.sh.OrbitDocsQuery(dbIncome, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		income := []Income{}

		if len(i) == 0 {
			log.Fatal(err)
		}

		err = json.Unmarshal([]byte(i), &income) // Unmarshal the byte slice directly
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.income = income[0]
			// check if there is a matching income year and month to current moment
			if w.income.Period.Year() == time.Now().Year() && w.income.Period.Month() == time.Now().Month() {
				w.userBalance.Balance = (w.userBalance.Balance + w.income.Amount)
				w.userBalance.Income = w.income.Amount
				w.userBalance.LastReceived = time.Now()
				ctx.SetState("balance", w.userBalance)
				w.updateBalance(ctx)
			}
		})
	})
}

func (w *wallet) doPay(ctx app.Context, e app.Event) {
	ctx.Navigate("payment")
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
						app.Span().Text(strconv.Itoa(w.userBalance.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().Class("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Monthly Recurring"),
							app.Span().Text(strconv.Itoa(w.userBalance.Income/100)+" GUBI"),
						),
					),
					app.Div().Class("lower-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("My Payment ID"),
							app.Span().Text(w.userID),
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
				app.Div().Class("menu-btn").Body(
					app.Button().Class("submit").Type("submit").Text("Pay").OnClick(w.doPay),
				),
			),
		),
	)
}
