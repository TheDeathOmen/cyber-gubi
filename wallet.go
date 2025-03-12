package main

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

const dbIncome = "income"
const dbUserBalance = "user_balance"
const dbInflation = "inflation"

// wallet is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type wallet struct {
	app.Compo
	sh           *shell.Shell
	loggedIn     bool
	isBusiness   bool
	userID       string
	userBalance  UserBalance
	income       Income
	transactions []Transaction
}

type UserBalance struct {
	ID           string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                     // Unique identifier for the user
	Balance      int    `mapstructure:"balance" json:"balance" validate:"uuid_rfc4122"`             // Balance of the user in cents
	Income       int    `mapstructure:"income" json:"income" validate:"uuid_rfc4122"`               // Recurring income of the user in cents
	LastReceived string `mapstructure:"last_received" json:"last_received" validate:"uuid_rfc4122"` // Date when basic income was last received
}

type Income struct {
	ID     string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`       // Unique identifier for the income
	Amount int    `mapstructure:"amount" json:"amount" validate:"uuid_rfc4122"` // Amount of the income in cents
	Period string `mapstructure:"period" json:"period" validate:"uuid_rfc4122"` // Period the income is valid for
}

func (w *wallet) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	w.sh = sh

	ctx.GetState("loggedIn", &w.loggedIn)
	if !w.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &w.userID)

	ctx.GetState("isBusiness", &w.isBusiness)

	log.Println("w.isBusiness", w.isBusiness)

	// w.updateIncome()
	// w.deleteIncome()
	// w.deleteBalances()
	// w.deleteTransactions()
	// w.deleteInflation()
	// return

	w.getBalance(ctx)
}

func daysRemainingInMonth(date time.Time) int {
	// Calculate the first day of the next month
	firstDayOfNextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, date.Location())

	// Subtract one day to get the last day of the current month
	lastDayOfMonth := firstDayOfNextMonth.Add(-time.Hour * 24)

	// Set the current date to midnight
	midnightToday := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	// Calculate the difference between the last day of the month and midnight today
	diff := lastDayOfMonth.Sub(midnightToday)

	// Convert the duration to days
	days := int(diff.Hours()/24) + 1 // Add 1 to include today

	return days
}

func (w *wallet) deleteTransactions() {
	err := w.sh.OrbitDocsDelete(dbTransaction, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deleteInflation() {
	err := w.sh.OrbitDocsDelete(dbInflation, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deleteBalances() {
	err := w.sh.OrbitDocsDelete(dbUserBalance, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) getTransactions(ctx app.Context) {
	ctx.Async(func() {
		t, err := w.sh.OrbitDocsQuery(dbTransaction, "sender_id,receiver_id", w.userID)
		if err != nil {
			log.Fatal(err)
		}

		transactions := []Transaction{}

		if len(t) != 0 {
			err = json.Unmarshal(t, &transactions) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			if len(transactions) > 0 {
				sort.Slice(transactions, func(i, j int) bool {
					return transactions[i].Timestamp.After(transactions[j].Timestamp)
				})

				w.transactions = append(w.transactions, transactions...)
			}
		})
	})
}

func (w *wallet) getInflation(ctx app.Context) {
	ctx.Async(func() {
		_, err := w.sh.OrbitDocsQuery(dbInflation, "all", "")
		if err != nil {
			log.Fatal(err)
		}
	})
}

func (w *wallet) getBalance(ctx app.Context) {
	ctx.Async(func() {
		b, err := w.sh.OrbitDocsQuery(dbUserBalance, "_id", w.userID)
		if err != nil {
			log.Fatal(err)
		}

		userBalances := []UserBalance{}

		if len(b) == 0 {
			ctx.Dispatch(func(ctx app.Context) {
				w.userBalance = UserBalance{}
				ctx.SetState("balance", w.userBalance)
				if !w.isBusiness {
					w.getIncome(ctx)
				} else {
					w.updateBalance(ctx)
				}
			})
			return
		} else {
			err = json.Unmarshal(b, &userBalances) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.userBalance = userBalances[0]
			ctx.SetState("balance", w.userBalance)

			// check if recurring income was received for this month
			if !w.isBusiness && w.userBalance.LastReceived != strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month())) {
				w.getIncome(ctx)
			} else {
				w.getTransactions(ctx)
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

		ctx.Dispatch(func(ctx app.Context) {
			w.getTransactions(ctx)
		})
	})
}

func (w *wallet) updateIncome() {
	income := &Income{
		ID:     uuid.NewString(),
		Amount: 100000,
		Period: strconv.Itoa(time.Now().Year()) + "/" + strconv.Itoa(int(time.Now().Month())),
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
			for _, inc := range income {
				if inc.Period == strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month())) {
					w.income = inc
				}
			}

			// check if there is a matching income year and month to current moment
			if w.income.Period == strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month())) {
				w.userBalance.Balance = (w.userBalance.Balance + w.income.Amount)
				w.userBalance.Income = w.income.Amount
				w.userBalance.LastReceived = strconv.Itoa(time.Now().Year()) + "/" + strconv.Itoa(int(time.Now().Month()))
				ctx.SetState("balance", w.userBalance)
				w.updateBalance(ctx)
			} else {
				w.getTransactions(ctx)
			}
		})
	})
}

func (w *wallet) goToPayments(ctx app.Context, e app.Event) {
	ctx.Navigate("payment")
}

func (w *wallet) showTransactionDetails(ctx app.Context, e app.Event) {
	ctx.JSSrc().Call("setAttribute", "style", "height: auto")
}

func (w *wallet) hideTransactionDetails(ctx app.Context, e app.Event) {
	ctx.JSSrc().Call("setAttribute", "style", "height: 55px")
}

// The Render method is where the component appearance is defined. Here, a
// wallet is displayed.
func (w *wallet) Render() app.UI {
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
						app.Span().Text(strconv.Itoa(w.userBalance.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
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
							app.Span().Class("span-body").Text(w.userID),
						),
					),
				),
				app.Div().Class("transactions").Body(
					app.Span().Class("t-desc").Text("Recent Transactions"),
					app.Range(w.transactions).Slice(func(i int) app.UI {
						return app.Div().Class("transaction").Body(
							app.Div().Class("t-details").Body(
								app.Div().Class("t-title").Body(
									app.If(w.transactions[i].SenderID == w.userID, func() app.UI {
										return app.Span().Text("Purchase ID: " + w.transactions[i].ID)
									}).Else(func() app.UI {
										return app.Span().Text("Sale ID: " + w.transactions[i].ID)
									}),
								),
								app.Div().Class("t-time").Body(
									app.Span().Text(w.transactions[i].Timestamp.Format("2006-01-02 15:04:05")),
								),
								app.Div().Class("t-more-details").Body(
									app.Div().Class("col-1").Body(
										app.Span().Text("Item"),
										app.Range(w.transactions[i].ProductsServices).Slice(func(n int) app.UI {
											return app.Span().Text(w.transactions[i].ProductsServices[n].Name)
										}),
									),
									app.Div().Class("col-2").Body(
										app.Span().Text("Amount"),
										app.Range(w.transactions[i].ProductsServices).Slice(func(n int) app.UI {
											return app.Span().Text(w.transactions[i].ProductsServices[n].Amount)
										}),
									),
									app.Div().Class("col-3").Body(
										app.Span().Text("Price"),
										app.Range(w.transactions[i].ProductsServices).Slice(func(n int) app.UI {
											return app.Span().Text(w.transactions[i].ProductsServices[n].Price)
										}),
									),
								),
							).OnMouseOver(w.showTransactionDetails).OnMouseLeave(w.hideTransactionDetails),
							app.Div().Class("t-price").Body(
								app.If(w.transactions[i].SenderID == w.userID, func() app.UI {
									return app.Span().Text("-" + strconv.Itoa(w.transactions[i].TotalCost/100) + " GUBI")
								}).Else(func() app.UI {
									return app.Span().Text("+" + strconv.Itoa(w.transactions[i].TotalCost/100) + " GUBI")
								}),
							),
						)
					}),
				),
				app.Div().Class("menu-btn").Body(
					app.Button().Class("submit").Type("submit").Text("Make a payment").OnClick(w.goToPayments),
				),
			),
		),
	)
}
