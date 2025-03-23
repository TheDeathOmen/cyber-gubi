package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	mathRand "math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	shell "github.com/stateless-minds/go-ipfs-api"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const dbUser = "user"

// auth is a component that uses webauthn and biometrics. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type auth struct {
	app.Compo
	sh                     *shell.Shell
	webAuthn               *webauthn.WebAuthn
	descriptorJSON         string
	currentUser            User
	country                string
	region                 string
	entity                 string
	termsAccepted          bool
	notificationPermission app.NotificationPermission
	businessName           string
	associateName          string
	newAssociateName       string
	vat                    string
}

// Credential represents the structure for credential information.
type Credential struct {
	ID            []byte        `mapstructure:"id" json:"id"`
	PublicKey     []byte        `mapstructure:"publicKey" json:"public_key"`
	Authenticator Authenticator `mapstructure:"authenticator" json:"authenticator"`
}

// Authenticator represents the authenticator details.
type Authenticator struct {
	AAGUID       []byte `mapstructure:"AAGUID" json:"aaguid"`
	Attachment   string `mapstructure:"attachment" json:"attachment"`
	CloneWarning bool   `mapstructure:"cloneWarning" json:"clone_warning"`
	SignCount    int    `mapstructure:"signCount" json:"sign_count"`
}

type User struct {
	ID            []byte                `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                       // Unique identifier for the user (should be a byte array)
	Name          string                `mapstructure:"name" json:"name" validate:"uuid_rfc4122"`                     // Username or identifier for the user
	DisplayName   string                `mapstructure:"display_name" json:"display_name" validate:"uuid_rfc4122"`     // Display name for the user
	CredentialIDs []webauthn.Credential `mapstructure:"credential_ids" json:"credential_ids" validate:"uuid_rfc4122"` // List of credential IDs associated with the user
	Descriptor    map[string][]float32  `mapstructure:"descriptor" json:"descriptor" validate:"uuid_rfc4122"`         // Face descriptor for the user
	VAT           string                `mapstructure:"vat" json:"vat" validate:"uuid_rfc4122"`                       // VAT when company
	Country       string                `mapstructure:"country" json:"country" validate:"uuid_rfc4122"`
	Region        string                `mapstructure:"region" json:"region" validate:"uuid_rfc4122"` // Country
}

// Define your own struct that matches the CredentialCreation structure
type MyCredentialCreation struct {
	Challenge []byte
	RP        RelyingParty
	User      User
}

type RelyingParty struct {
	Name string
	ID   string
}

type UserVerification struct {
	UserVerificationRequirement string
}

// PublicKeyCredentialType represents the type of public key credential.
type PublicKeyCredentialType string

const (
	PublicKeyCredentialTypePublicKey PublicKeyCredentialType = "public-key"
)

// Parameters represents the parameters for public key credentials.
type Parameters struct {
	Type PublicKeyCredentialType `json:"type"` // Type of credential (e.g., "public-key")
	Alg  int                     `json:"alg"`  // COSE algorithm identifier
}

// RegistrationData represents the structure of the registration data
type RegistrationData struct {
	ID       string         `json:"id"`
	RawID    string         `json:"rawId"`
	Type     string         `json:"type"`
	Response ResponseCreate `json:"response"`
}

// RegistrationData represents the structure of the registration data
type LoginData struct {
	ID       string      `json:"id"`
	RawID    string      `json:"rawId"`
	Type     string      `json:"type"`
	Response ResponseGet `json:"response"`
}

type ResponseCreate struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AttestationObject string `json:"attestationObject"`
}

type ResponseGet struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
}

// Implementing the webauthn.User interface
func (u *User) WebAuthnID() []byte {
	return u.ID
}

func (u *User) WebAuthnName() string {
	return u.Name
}

func (u *User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.CredentialIDs
}

// Methods to manage credentials
func (u *User) AddCredential(credential webauthn.Credential) {
	u.CredentialIDs = append(u.CredentialIDs, credential)
}

func (u *User) UpdateCredential(credential webauthn.Credential) {
	for i, cred := range u.CredentialIDs {
		if string(cred.ID) == string(credential.ID) {
			u.CredentialIDs[i] = credential // Update existing credential ID
			break
		}
	}
}

func (a *auth) OnMount(ctx app.Context) {
	a.notificationPermission = ctx.Notifications().Permission()
	if a.notificationPermission == "default" {
		a.notificationPermission = ctx.Notifications().RequestPermission()
	}

	sh := shell.NewShell("localhost:5001")
	a.sh = sh

	a.findCountry(ctx)

	// a.deleteUsers()
	// return

	wconfig := &webauthn.Config{
		RPDisplayName: "cyber-gubi",                      // Display Name for your site
		RPID:          "localhost",                       // Generally the FQDN for your site
		RPOrigins:     []string{"http://localhost:8000"}, // Allowed origins for WebAuthn requests
	}

	var err error

	if a.webAuthn, err = webauthn.New(wconfig); err != nil {
		ctx.Notifications().New(app.Notification{
			Title: "Webauthn instantiate error",
			Body:  err.Error(),
		})
		log.Fatal(err)
	}

	a.fetchUser(ctx)

	ctx.ObserveState("entity", &a.entity)

	ctx.ObserveState("termsAccepted", &a.termsAccepted).
		OnChange(func() {
			if a.entity == "individual" {
				a.beginRegistration(ctx)
			}
		})

	ctx.ObserveState("vat", &a.vat).
		OnChange(func() {
			if a.entity == "business" && a.termsAccepted {
				ctx.GetState("businessName", &a.businessName)
				ctx.GetState("associateName", &a.associateName)
				duplicatesFound := a.checkForDuplicates(ctx)
				if !duplicatesFound {
					a.beginRegistration(ctx)
				}
			}
		})
}

func (a *auth) findCountry(ctx app.Context) {
	myPeer, err := a.sh.ID()
	if err != nil {
		log.Fatal(err)
	}

	for _, addr := range myPeer.Addresses {
		// Split the address to handle multiaddr format
		ip, err := extractIP(addr)
		if err != nil {
			continue
		}
		if strings.Contains(ip, ":") {
			continue
		}
		if isPublicIP(ip) {
			fmt.Println("Potential public IP:", ip)
			ctx.Async(func() {
				r, err := http.Get("http://ip-api.com/json/" + ip + "?fields=countryCode,region")
				if err != nil {
					log.Fatal(err)
				}
				defer r.Body.Close()

				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Fatal(err)
				}

				var info map[string]interface{}

				err = json.Unmarshal(b, &info)
				if err != nil {
					log.Fatal(err)
				}

				// Storing HTTP response in component field:
				ctx.Dispatch(func(ctx app.Context) {
					a.country = info["countryCode"].(string)
					ctx.SetState("country: ", a.country)

					a.region = info["region"].(string)
					ctx.SetState("region: ", a.region)
				})
			})
		}
	}
}

func isPublicIP(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr != nil {
		// Check if the IP is not a private or loopback address
		if ipAddr.IsPrivate() || ipAddr.IsLoopback() {
			return false
		}
		return true
	}
	return false
}

func extractIP(addr string) (string, error) {
	// Simple function to extract IP from multiaddr format
	// This might need adjustments based on the actual format of addr
	parts := strings.Split(addr, "/")
	for _, part := range parts {
		if net.ParseIP(part) != nil {
			return part, nil
		}
	}
	return "", fmt.Errorf("no IP found in address")
}

func (a *auth) getIncome(ctx app.Context) {
	ctx.Async(func() {
		i, err := a.sh.OrbitDocsQuery(dbIncome, "all", "")
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
			doIndexer := true
			for _, inc := range income {
				if inc.Period == strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month()+1)) {
					doIndexer = false
				}
			}

			if doIndexer {
				a.sh.RunInflationIndexer()
			}
		})
	})
}

// Function to generate a new user
func NewUser() (*User, error) {
	return &User{
		ID:            protocol.URLEncodedBase64(uuid.NewString()),
		CredentialIDs: []webauthn.Credential{}, // Initialize with no credentials
	}, nil
}

func (a *auth) doRegister(ctx app.Context, e app.Event) {
	a.descriptorJSON = e.Get("detail").Get("descriptor").String()
	ctx.GetState("newAssociateName", &a.newAssociateName)

	if len(a.newAssociateName) > 0 {
		a.updateUser(ctx)
	} else {
		err := a.getUser(ctx)
		if err != nil {
			app.Window().GetElementByID("main-menu").Call("click")
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Error",
				Body:  "You can not register another person on this device with the same private keys. Clone https://github.com/stateless-minds/cyber-gubi-local and run it for the new registration.",
			})
		}
	}
}

func (a *auth) doLogin(ctx app.Context, e app.Event) {
	a.descriptorJSON = e.Get("detail").Get("descriptor").String()
	if len(a.descriptorJSON) == 0 {
		log.Fatal("descriptorJSON is empty")
	}

	var descriptor map[string][]float32

	err := json.Unmarshal([]byte(a.descriptorJSON), &descriptor)
	if err != nil {
		log.Fatal(err)
	}

	for name := range descriptor {
		if len(a.currentUser.Descriptor[name]) > 0 {
			ctx.SetState("userID", string(a.currentUser.ID))
			if len(a.currentUser.VAT) > 0 {
				ctx.SetState("isBusiness", true)
				ctx.SetState("businessName", a.currentUser.Name)
				ctx.SetState("associateName", name)
			}
			a.beginLogin(ctx, string(a.currentUser.CredentialIDs[0].ID))
		}
	}
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

func (a *auth) fetchUser(ctx app.Context) {
	descriptor := map[string][]float32{}
	var descriptorJSON []byte
	err := a.getUser(ctx)
	if err != nil {
		log.Println(err)
		descriptorJSON, err = json.Marshal(descriptor)
	} else {
		descriptorJSON, err = json.Marshal(a.currentUser.Descriptor)
	}

	if err != nil {
		log.Fatal(err)
	}

	// Send response event to child
	app.Window().Get("parent").Get("window").Call("dispatchEvent", // Target the iframe's window
		app.Window().Get("CustomEvent").New("descriptorsFetched", map[string]interface{}{
			"detail": map[string]interface{}{
				"descriptors": string(descriptorJSON),
			},
		}),
	)

	days := daysRemainingInMonth(time.Now())
	if days <= 3 {
		a.getIncome(ctx)
	}
}

func (a *auth) getUser(ctx app.Context) error {
	res, err := a.sh.OrbitDocsQueryEnc(dbUser, "own", "")
	if err != nil {
		log.Fatal(err)
	}

	// sanitize json
	res1 := strings.ReplaceAll(string(res), "\\", "")

	res2 := strings.ReplaceAll(res1, `""`, `"`)

	res3 := strings.ReplaceAll(res2, `"[`, `[`)

	res4 := strings.ReplaceAll(res3, `:",`, `:"",`)

	res5 := strings.ReplaceAll(res4, `]"`, `]`)

	res6 := strings.ReplaceAll(res5, `"{`, `{`)

	res7 := strings.ReplaceAll(res6, `}"`, `}`)

	users := []User{}

	if len(res) == 0 {
		return errors.New("no user found")
	}

	err = json.Unmarshal([]byte(res7), &users)
	if err != nil {
		return err
	}

	a.currentUser = users[0]
	ctx.SetState("currentUser", a.currentUser)

	return nil
}

func (a *auth) deleteUsers() {
	err := a.sh.OrbitDocsDelete(dbUser, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) createUser(ctx app.Context, userID, credentialID string) {
	ctx.Async(func() {
		var descriptor []float32
		err := json.Unmarshal([]byte(a.descriptorJSON), &descriptor)
		if err != nil {
			log.Fatal(err)
		}

		descriptorMap := make(map[string][]float32)
		if len(a.associateName) == 0 {
			// pseudonymous for individuals
			descriptorMap["user"] = descriptor
		} else {
			descriptorMap[a.associateName] = descriptor
			ctx.SetState("associateName", &a.associateName)
		}

		user := User{
			Name:        a.businessName,
			DisplayName: a.businessName,
			ID:          protocol.URLEncodedBase64(userID),
			CredentialIDs: []webauthn.Credential{
				{
					ID: []byte(credentialID),
				},
			},
			Descriptor: descriptorMap,
			VAT:        a.vat,
			Country:    a.country,
			Region:     a.region,
		}

		userJSON, err := json.Marshal(user)
		if err != nil {
			log.Fatal(err)
		}

		err = a.sh.OrbitDocsPutEnc(dbUser, userJSON)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			a.currentUser = user
		})
	})
}

func (a *auth) updateUser(ctx app.Context) {
	ctx.Async(func() {
		var descriptor []float32
		err := json.Unmarshal([]byte(a.descriptorJSON), &descriptor)
		if err != nil {
			log.Fatal(err)
		}
		a.currentUser.Descriptor[a.newAssociateName] = descriptor

		userJSON, err := json.Marshal(a.currentUser)
		if err != nil {
			log.Fatal(err)
		}

		err = a.sh.OrbitDocsPutEnc(dbUser, userJSON)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			ctx.DelState("newAssociateName")
			ctx.Notifications().New(app.Notification{
				Title: "Success",
				Body:  "You have added associate " + a.newAssociateName + ". Any of you can log in now.",
			})
			ctx.Reload()
		})
	})
}

func (a *auth) checkForDuplicates(ctx app.Context) bool {
	res, err := a.sh.OrbitDocsQueryEnc(dbUser, "all", "")
	if err != nil {
		log.Fatal(err)
	}

	users := []map[string]interface{}{}

	if len(res) != 0 {
		err = json.Unmarshal([]byte(res), &users)
		if err != nil {
			log.Fatal(err)
		}
	}

	duplicates := false

	if len(users) > 0 {
		for _, usrs := range users {
			for k, v := range usrs {
				if k == "name" || k == "display_name" {
					if v == a.businessName {
						ctx.Notifications().New(app.Notification{
							Title: "Registration error",
							Body:  "Business with this name already exists.",
						})
						duplicates = true
						break
					}
				} else if k == "vat" {
					if v == a.vat {
						ctx.Notifications().New(app.Notification{
							Title: "Registration error",
							Body:  "Business with this VAT number already exists.",
						})
						duplicates = true
						break
					}
				}
			}
		}
	}

	return duplicates
}

func (a *auth) beginRegistration(ctx app.Context) {
	// RelyingParty instance
	relyingParty := RelyingParty{
		Name: a.webAuthn.Config.RPDisplayName,
		ID:   a.webAuthn.Config.RPID,
	}

	userID := uuid.NewString()

	us := User{
		ID: []byte(userID),
	}

	if len(a.businessName) > 0 {
		us.Name = a.businessName
		us.DisplayName = a.businessName
	}

	rp := app.ValueOf(map[string]interface{}{
		"name": relyingParty.Name,
		"id":   relyingParty.ID,
	})

	usr := app.ValueOf(map[string]interface{}{
		"name":        us.Name,
		"displayName": us.DisplayName,
	})

	usID := app.Window().Get("Uint8Array").New(len(us.ID))

	usr.Set("id", usID)

	as := app.ValueOf(map[string]interface{}{
		"authenticatorAttachment": "platform",
		"userVerification":        "required",
		"residentKey":             "required",
	})

	// Create pubKeyCredParams as an array in JavaScript
	pubKeyCredParams := app.Window().Get("Array").New()

	// Add parameters to the pubKeyCredParams array
	param1 := app.Window().Get("Object").New()
	param1.Set("type", "public-key")
	param1.Set("alg", -7) // Example algorithm identifier for ES256
	pubKeyCredParams.Call("push", param1)

	param2 := app.Window().Get("Object").New()
	param2.Set("type", "public-key")
	param2.Set("alg", -257) // Example algorithm identifier for RS256
	pubKeyCredParams.Call("push", param2)

	// Generate a random challenge
	// Create a new random source seeded with the current time
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	challengeByteArray := make([]byte, 32)
	r.Read(challengeByteArray)
	challenge := app.Window().Get("Uint8Array").New(len(challengeByteArray)) // Generate a random challenge
	// Fill the Uint8Array with the byte values
	for i, b := range challengeByteArray {
		challenge.SetIndex(i, b)
	}

	obj := app.Window().Get("Object").New()
	obj.Set("challenge", challenge)
	obj.Set("rp", rp)
	obj.Set("user", usr)
	obj.Set("pubKeyCredParams", pubKeyCredParams)
	obj.Set("authenticatorSelection", as)
	obj.Set("publicKey", obj)

	// Access the navigator object
	promise := app.Window().Get("navigator").Get("credentials").Call("create", obj)

	// Step 3: Handle the promise response
	promise.Call("then", app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		if len(args) > 0 {
			cred := args[0] // The PublicKeyCredential object
			// Get the credentialId
			credentialID := cred.Get("id").String()
			a.createUser(ctx, userID, credentialID)
			ctx.SetState("userID", userID)
			if len(a.vat) > 0 {
				ctx.SetState("isBusiness", true)
			}
			a.beginLogin(ctx, credentialID)
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "No credential returned.",
			})
		}
		return nil
	})).Call("catch", app.FuncOf(func(this app.Value, p []app.Value) interface{} {
		if len(p) > 0 {
			err := p[0]
			// Attempt to read the error message
			var errorMessage string
			if err.Get("message").String() != "" {
				errorMessage = err.Get("message").String() // Standard way to get the message
			} else if err.Get("error").String() != "" {
				errorMessage = err.Get("error").String() // Some errors might have an 'error' property
			} else {
				errorMessage = "Unknown error occurred."
			}

			// Notify user through UI instead of terminating application
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "Credential creation failed: " + errorMessage,
			})
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "Unknown error occurred.",
			})
		}

		return nil
	}))
}

func (a *auth) beginLogin(ctx app.Context, credentialID string) {
	// Generate a random challenge
	// Create a new random source seeded with the current time
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	challengeByteArray := make([]byte, 32)
	r.Read(challengeByteArray)
	challenge := app.Window().Get("Uint8Array").New(len(challengeByteArray)) // Generate a random challenge
	// Fill the Uint8Array with the byte values
	for i, b := range challengeByteArray {
		challenge.SetIndex(i, b)
	}

	// Create the allowCredentials array
	allowCredentials := app.Window().Get("Array").New(0) // Start with an empty array

	// Create the credential descriptor object
	credDescriptor := app.Window().Get("Object").New()
	credDescriptor.Set("type", "public-key")

	// Convert credentialID to Uint8Array (if it isn't already)
	// Assuming credentialID is a *string* representation of the ID, if not then this conversion is not needed
	encoder := app.Window().Get("TextEncoder").New()
	credentialIDUint8Array := encoder.Call("encode", credentialID)
	credDescriptor.Set("id", credentialIDUint8Array)

	// Add the credential descriptor to the allowCredentials array
	allowCredentials.Call("push", credDescriptor)
	obj := app.Window().Get("Object").New()
	obj.Set("challenge", challenge)
	obj.Set("rpId", "localhost")
	obj.Set("userVerification", "required")
	obj.Set("allowCredentials:", allowCredentials)
	obj.Set("publicKey", obj)

	// Access the navigator object
	promise := app.Window().Get("navigator").Get("credentials").Call("get", obj)

	// Step 3: Handle the promise response
	promise.Call("then", app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		if len(args) > 0 {
			ctx.Notifications().New(app.Notification{
				Title: "Success",
				Body:  "Login successful!",
			})
			ctx.SetState("loggedIn", true)
			// redirect to wallet
			ctx.Navigate("/wallet")
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "No credential returned.",
			})
			log.Fatal("No credential returned")
		}
		return nil
	})).Call("catch", app.FuncOf(func(this app.Value, p []app.Value) interface{} {
		if len(p) > 0 {
			err := p[0]
			// Attempt to read the error message
			var errorMessage string
			if err.Get("message").String() != "" {
				errorMessage = err.Get("message").String() // Standard way to get the message
			} else if err.Get("error").String() != "" {
				errorMessage = err.Get("error").String() // Some errors might have an 'error' property
			} else {
				errorMessage = "Unknown error occurred."
			}

			// Notify user through UI instead of terminating application
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "Credential fetching failed: " + errorMessage,
			})
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "Unknown error occurred.",
			})
		}

		return nil
	}))
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (a *auth) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Authentication"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card card-auth").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Face ID"),
						),
					),
					app.Div().Class("lower-row").Body(
						app.Div().Class("card-item").Body(
							app.Div().Class("container").Body(
								app.Video().ID("video").Width(225).Height(225).AutoPlay(true).Muted(true),
								app.Canvas().ID("canvas").Width(225).Height(225),
							),
						),
					),
				),
				app.Div().Class("drawer drawer-auth").Body(
					app.Div().ID("auth-bar").Class("auth-bar").Body(
						app.Span().Class("auth-message").Text("Authenticating"),
						app.Span().Class("blinking").Text("..."),
					),
					app.Input().ID("register-btn").OnClick(a.doRegister).Hidden(true),
					app.Input().ID("login-btn").OnClick(a.doLogin).Hidden(true),
				),
			),
		),
	)
}
