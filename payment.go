package main

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

const dbTransaction = "transaction"

// payment is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type payment struct {
	app.Compo
	sh           *shell.Shell
	loggedIn     bool
	userID       string
	userBalance  UserBalance
	userBalances []UserBalance
}

type Transaction struct {
	ID         string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                 // Unique identifier for the transaction
	SenderID   string `mapstructure:"sender_id" json:"sender_id" validate:"uuid_rfc4122"`     // Sender user id
	ReceiverID string `mapstructure:"receiver_id" json:"receiver_id" validate:"uuid_rfc4122"` // Recipient user id
	ProductOrService
	Timestamp time.Time `mapstructure:"timestamp" json:"timestamp" validate:"uuid_rfc4122"` // Timestamp of the transaction
}

type ProductOrService struct {
	ID     string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"` // Unique identifier for the product
	Name   string `mapstructure:"name" json:"name" validate:"uuid_rfc4122"`
	Price  int    `mapstructure:"price" json:"price" validate:"uuid_rfc4122"`
	Amount int    `mapstructure:"amount" json:"amount" validate:"uuid_rfc4122"`
}

func (p *payment) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	p.sh = sh

	ctx.GetState("loggedIn", &p.loggedIn)
	if !p.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &p.userID)
	ctx.GetState("balance", &p.userBalance)
	p.getBalances(ctx)
}

func (p *payment) getBalance(userID string) (balance UserBalance, err error) {
	b, err := p.sh.OrbitDocsQuery(dbUserBalance, "_id", userID)
	if err != nil {
		return UserBalance{}, err
	}

	if len(b) == 0 {
		return UserBalance{}, err
	}

	userBalances := []UserBalance{}

	err = json.Unmarshal(b, &userBalances) // Unmarshal the byte slice directly
	if err != nil {
		return UserBalance{}, err
	}

	return userBalances[0], nil
}

func removeSelfFromUserResults(userBalances []UserBalance, userID string) []UserBalance {
	for i, ub := range userBalances {
		if ub.ID == userID {
			return append(userBalances[:i], userBalances[i+1:]...)
		}
	}
	return userBalances
}

func (p *payment) getBalances(ctx app.Context) {
	ctx.Async(func() {
		b, err := p.sh.OrbitDocsQuery(dbUserBalance, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		if len(b) == 0 {
			log.Fatal(err)
		}

		userBalances := []UserBalance{}

		err = json.Unmarshal(b, &userBalances) // Unmarshal the byte slice directly
		if err != nil {
			log.Fatal(err)
		}

		userBalances = removeSelfFromUserResults(userBalances, p.userID)

		ctx.Dispatch(func(ctx app.Context) {
			p.userBalances = userBalances
		})

	})
}

func (p *payment) updateBalance(userID string, balance, income int, timestamp time.Time) error {
	userBalance := UserBalance{
		ID:           userID,
		Balance:      balance,
		Income:       income,
		LastReceived: timestamp,
	}

	userBalanceJSON, err := json.Marshal(userBalance)
	if err != nil {
		return err
	}

	err = p.sh.OrbitDocsPut(dbUserBalance, userBalanceJSON)
	if err != nil {
		return err
	}

	return nil
}

func (p *payment) storeTransaction(transaction Transaction) error {
	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	err = p.sh.OrbitDocsPut(dbTransaction, transactionJSON)
	if err != nil {
		return err
	}

	return nil
}

func (p *payment) showProduct(ctx app.Context, e app.Event) {
	e.PreventDefault()
	app.Window().GetElementByID("service-name").Call("removeAttribute", "required")
	app.Window().GetElementByID("service-amount").Call("removeAttribute", "required")
	app.Window().GetElementByID("tab-service").Get("classList").Call("remove", "tab-active")
	ctx.JSSrc().Get("classList").Call("add", "tab-active")
	// Hide Service Tab and show Product Tab
	app.Window().GetElementByID("product-tab").Call("setAttribute", "style", "display: block")
	app.Window().GetElementByID("service-tab").Call("setAttribute", "style", "display: none")
}

func (p *payment) showService(ctx app.Context, e app.Event) {
	e.PreventDefault()
	app.Window().GetElementByID("product-name").Call("removeAttribute", "required")
	app.Window().GetElementByID("product-price").Call("removeAttribute", "required")
	app.Window().GetElementByID("product-amount").Call("removeAttribute", "required")
	app.Window().GetElementByID("tab-product").Get("classList").Call("remove", "tab-active")
	ctx.JSSrc().Get("classList").Call("add", "tab-active")
	// Hide Product Tab and show Service Tab
	app.Window().GetElementByID("product-tab").Call("setAttribute", "style", "display: none; ")
	app.Window().GetElementByID("service-name").Call("setAttribute", "required", true)
	app.Window().GetElementByID("service-amount").Call("setAttribute", "required", true)
	app.Window().GetElementByID("service-tab").Call("setAttribute", "style", "display: block")
}

func (p *payment) doPayment(ctx app.Context, e app.Event) {
	e.PreventDefault()
	valid := app.Window().GetElementByID("pay-form").Call("reportValidity").Bool()
	if valid {
		tabActive := app.Window().Get("document").Call("getElementsByClassName", "tab-active").Index(0).Get("value").String()
		paymentID := app.Window().GetElementByID("payment-id").Get("value").String()
		transaction := Transaction{}
		transaction.ID = uuid.NewString()
		transaction.SenderID = p.userID
		transaction.ReceiverID = paymentID
		transaction.Timestamp = time.Now()
		if tabActive == "product" {
			productName := app.Window().GetElementByID("product-name").Get("value").String()
			productPriceInt, err := strconv.Atoi(app.Window().GetElementByID("product-price").Get("value").String())
			if err != nil {
				log.Fatal(err)
			}
			productPrice := productPriceInt * 100 // in cents
			productAmountInt, err := strconv.Atoi(app.Window().GetElementByID("product-amount").Get("value").String())
			if err != nil {
				log.Fatal(err)
			}
			productAmount := productAmountInt
			transaction.ProductOrService.Name = productName
			transaction.ProductOrService.Price = productPrice
			transaction.ProductOrService.Amount = productAmount
		} else if tabActive == "service" {
			serviceName := app.Window().GetElementByID("service-name").Get("value").String()
			servicePriceInt, err := strconv.Atoi(app.Window().GetElementByID("service-price").Get("value").String())
			if err != nil {
				log.Fatal(err)
			}
			servicePrice := servicePriceInt * 100 // in cents
			serviceAmountInt, err := strconv.Atoi(app.Window().GetElementByID("service-amount").Get("value").String())
			if err != nil {
				log.Fatal(err)
			}
			serviceAmount := serviceAmountInt // full hours only
			transaction.ProductOrService.Name = serviceName
			transaction.ProductOrService.Price = servicePrice
			transaction.ProductOrService.Amount = serviceAmount
		} else {
			log.Fatal(errors.New("no tab selected"))
		}

		// update sender balance
		err := p.updateBalance(p.userID, p.userBalance.Balance-(transaction.ProductOrService.Price*transaction.ProductOrService.Amount), p.userBalance.Income, p.userBalance.LastReceived)
		if err != nil {
			log.Fatal(err)
		}
		// get receiver balance
		receiverBalance, err := p.getBalance(transaction.ReceiverID)
		if err != nil {
			log.Fatal(err)
		}
		// update receiver balance
		err = p.updateBalance(transaction.ReceiverID, receiverBalance.Balance+(transaction.ProductOrService.Price*transaction.ProductOrService.Amount), receiverBalance.Income, receiverBalance.LastReceived)
		if err != nil {
			// rollback sender balance
			err := p.updateBalance(p.userID, p.userBalance.Balance+(transaction.ProductOrService.Price*transaction.ProductOrService.Amount), p.userBalance.Income, p.userBalance.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		// store transaction
		err = p.storeTransaction(transaction)
		if err != nil {
			// rollback sender balance
			err = p.updateBalance(p.userID, p.userBalance.Balance+(transaction.ProductOrService.Price*transaction.ProductOrService.Amount), p.userBalance.Income, p.userBalance.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			// rollback receiver balance
			err = p.updateBalance(transaction.ReceiverID, receiverBalance.Balance-(transaction.ProductOrService.Price*transaction.ProductOrService.Amount), receiverBalance.Income, receiverBalance.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		app.Window().Get("alert").Invoke("payment successful!")
	}
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (p *payment) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
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
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Make a payment"),
							app.Form().ID("pay-form").Body(
								app.Label().For("payment-id").Text("Payment ID:"),
								app.Select().ID("payment-id").Name("payment-id").Body(
									app.Range(p.userBalances).Slice(func(i int) app.UI {
										return app.Option().Value(p.userBalances[i].ID).Text(p.userBalances[i].ID)
									}),
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
											Value("product").
											OnClick(p.showProduct),
										app.Button().
											ID("tab-service").
											Class("tab-button").
											Text("Service").
											Value("service").
											OnClick(p.showService),
									),

								// Product Tab Content
								app.Div().
									ID("product-tab").
									Class("tab-content").
									Body(
										app.Input().ID("product-name").Type("text").Name("product-name").Placeholder("Product name").Required(true),
										app.Input().ID("product-price").Type("number").Name("product-price").Placeholder("Single price").Required(true),
										app.Input().ID("product-amount").Type("number").Name("product-amount").Step(1).Placeholder("Number of products").Required(true),
									),

								// Service Tab Content
								app.Div().
									ID("service-tab").
									Class("tab-content").
									Body(
										app.Input().ID("service-name").Type("text").Name("service-name").Placeholder("Service name"),
										app.Input().ID("service-price").Type("number").Name("service-price").Placeholder("Price per hour"),
										app.Input().ID("service-amount").Type("number").Name("service-amount").Step(1).Placeholder("Number of hours"),
									).Hidden(true),
								app.Div().Class("drawer drawer-pay").Body(
									app.Div().Class("menu-btn").Body(
										app.Button().Class("submit").Type("submit").Text("Pay").OnClick(p.doPayment),
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
