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

// supplier is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type supplier struct {
	app.Compo
	sh            *shell.Shell
	loggedIn      bool
	userID        string
	userBalance   UserBalance
	plans         []Plan
	subscriptions []Subscription
	subscribed    bool
}

func (s *supplier) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	s.sh = sh

	ctx.GetState("loggedIn", &s.loggedIn)
	if !s.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &s.userID)
	ctx.GetState("balance", &s.userBalance)

	s.getPlans(ctx)
}

func (s *supplier) getPlans(ctx app.Context) {
	ctx.Async(func() {
		p, err := s.sh.OrbitDocsQuery(dbPlan, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		plans := []Plan{}

		if len(p) != 0 {
			err = json.Unmarshal(p, &plans) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		excludingOwnPlan := []Plan{}

		for _, plan := range plans {
			if plan.CreatedBy != s.userID {
				excludingOwnPlan = append(excludingOwnPlan, plan)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			s.plans = excludingOwnPlan
			s.deleteExpiredSubscriptions(ctx)
		})
	})
}

func (s *supplier) getSubscriptions(ctx app.Context) {
	ctx.Async(func() {
		subs, err := s.sh.OrbitDocsQuery(dbSubscription, "user_id", s.userID)
		if err != nil {
			log.Fatal(err)
		}

		subscriptions := []Subscription{}

		if len(subs) != 0 {
			err = json.Unmarshal(subs, &subscriptions) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			s.subscriptions = subscriptions
		})
	})
}

func (s *supplier) getBalance(userID string) (balance UserBalance, err error) {
	b, err := s.sh.OrbitDocsQuery(dbUserBalance, "_id", userID)
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

func (s *supplier) updateBalance(userID string, balance, income int, date string) error {
	userBalance := UserBalance{
		ID:           userID,
		Balance:      balance,
		Income:       income,
		LastReceived: date,
	}

	userBalanceJSON, err := json.Marshal(userBalance)
	if err != nil {
		return err
	}

	err = s.sh.OrbitDocsPut(dbUserBalance, userBalanceJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *supplier) storeTransaction(transaction Transaction) error {
	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	err = s.sh.OrbitDocsPut(dbTransaction, transactionJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *supplier) storeSubscription(subscription Subscription) error {
	subscriptionJSON, err := json.Marshal(subscription)
	if err != nil {
		return err
	}

	err = s.sh.OrbitDocsPut(dbSubscription, subscriptionJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *supplier) deleteExpiredSubscriptions(ctx app.Context) {
	ctx.Async(func() {
		s.sh.DeleteExpiredSubscriptions()

		ctx.Dispatch(func(ctx app.Context) {
			s.getSubscriptions(ctx)
		})
	})
}

func (s *supplier) deleteSubscription(id string) error {
	err := s.sh.OrbitDocsDelete(dbSubscription, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *supplier) doSubscribe(ctx app.Context, e app.Event) {
	e.PreventDefault()
	pid := ctx.JSSrc().Get("value").String()
	planID, err := strconv.Atoi(pid)
	if err != nil {
		log.Fatal(err)
	}

	plan := s.plans[planID]

	subscription := Subscription{
		ID:        uuid.NewString(),
		PlanID:    plan.ID,
		UserID:    s.userID,
		Price:     plan.Price,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 1, 0),
	}

	// store subscription
	s.storeSubscription(subscription)

	// store transaction
	transaction := Transaction{}
	transaction.ID = uuid.NewString()
	transaction.SenderID = s.userID
	transaction.ReceiverID = plan.CreatedBy
	transaction.Timestamp = time.Now()
	transaction.Date = strconv.Itoa(time.Now().Year()) + "/" + strconv.Itoa(int(time.Now().Month()))
	transaction.ProductsServices = []ProductService{
		{
			ID:     plan.ID,
			Name:   plan.Name,
			Price:  plan.Price,
			Amount: 1,
		},
	}
	transaction.TotalCost = plan.Price

	if s.userBalance.Balance-transaction.TotalCost < 0 {
		ctx.Notifications().New(app.Notification{
			Title: "Error",
			Body:  "Not enough funds.",
		})
		return
	}
	// update sender balance
	err = s.updateBalance(s.userID, s.userBalance.Balance-transaction.TotalCost, s.userBalance.Income, s.userBalance.LastReceived)
	if err != nil {
		log.Fatal(err)
	}
	// get receiver balance
	receiverBalance, err := s.getBalance(transaction.ReceiverID)
	if err != nil {
		log.Fatal(err)
	}
	// update receiver balance
	err = s.updateBalance(transaction.ReceiverID, receiverBalance.Balance+transaction.TotalCost, receiverBalance.Income, receiverBalance.LastReceived)
	if err != nil {
		// rollback sender balance
		err := s.updateBalance(s.userID, s.userBalance.Balance+transaction.TotalCost, s.userBalance.Income, s.userBalance.LastReceived)
		if err != nil {
			log.Fatal(err)
		}
		err = s.deleteSubscription(subscription.ID)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	// store transaction
	err = s.storeTransaction(transaction)
	if err != nil {
		// rollback sender balance
		err = s.updateBalance(s.userID, s.userBalance.Balance+transaction.TotalCost, s.userBalance.Income, s.userBalance.LastReceived)
		if err != nil {
			log.Fatal(err)
		}
		// rollback receiver balance
		err = s.updateBalance(transaction.ReceiverID, receiverBalance.Balance-transaction.TotalCost, receiverBalance.Income, receiverBalance.LastReceived)
		if err != nil {
			log.Fatal(err)
		}
		err = s.deleteSubscription(subscription.ID)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	s.userBalance.Balance = s.userBalance.Balance - transaction.TotalCost
	s.subscriptions = append(s.subscriptions, subscription)
	ctx.Update()

	ctx.Notifications().New(app.Notification{
		Title: "Success",
		Body:  "Subscription successful!",
	})
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (s *supplier) Render() app.UI {
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
						app.Span().Text(strconv.Itoa(s.userBalance.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header-sub").Text("Suppliers"),
						),
					),
				),
				app.Div().Class("subscriptions").Body(
					app.If(len(s.plans) == 0, func() app.UI {
						return app.Div().Class("subscription").Body(
							app.Span().Class("empty").Text("No plans yet"),
						).Style("pointer-events", "none")
					}),
					app.Range(s.plans).Slice(func(i int) app.UI {
						s.subscribed = false
						return app.Div().Class("subscription").Body(
							app.Div().Class("s-details").Body(
								app.Div().Class("s-title").Body(
									app.Span().Text(s.plans[i].Name),
								),
								app.Div().Class("s-time").Body(
									app.If(len(s.subscriptions) > 0, func() app.UI {
										return app.Range(s.subscriptions).Slice(func(n int) app.UI {
											return app.If(s.subscriptions[n].PlanID == s.plans[i].ID && s.subscriptions[n].UserID == s.userID, func() app.UI {
												return app.If(time.Now().Before(s.subscriptions[n].EndDate), func() app.UI {
													s.subscribed = true
													return app.Div().Class("menu-btn menu-sub menu-subscribed").Body(
														app.Button().Class("submit submit-sub").Type("submit").Text("Subscribed").Disabled(true),
													)
												})
											})
										})
									}),
									app.If(!s.subscribed, func() app.UI {
										return app.Div().Class("menu-btn menu-sub").Body(
											app.Button().Class("submit submit-sub").Type("submit").Text("Subscribe").Value(i).OnClick(s.doSubscribe),
										)
									}),
								),
							),
							app.Div().Class("s-price").Body(
								app.Span().Text(strconv.Itoa(s.plans[i].Price/100)+" GUBI"),
							),
						)
					}),
				),
			),
		),
	)
}
