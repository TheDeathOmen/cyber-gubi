package main

import (
	"encoding/json"
	"log"
	mathRand "math/rand"
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
	user                   *User
	users                  []*User
	entity                 string
	termsAccepted          bool
	notificationPermission app.NotificationPermission
	businessName           string
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
	Descriptor    []float32             `mapstructure:"descriptor" json:"descriptor" validate:"uuid_rfc4122"`         // Face descriptor for the user
	VAT           string                `mapstructure:"vat" json:"vat" validate:"uuid_rfc4122"`                       // VAT when company
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

	// a.deleteUsers()
	// return

	var err error
	wconfig := &webauthn.Config{
		RPDisplayName: "cyber-gubi",                      // Display Name for your site
		RPID:          "localhost",                       // Generally the FQDN for your site
		RPOrigins:     []string{"http://localhost:8000"}, // Allowed origins for WebAuthn requests
	}

	if a.webAuthn, err = webauthn.New(wconfig); err != nil {
		ctx.Notifications().New(app.Notification{
			Title: "Webauthn instantiate error",
			Body:  err.Error(),
		})
		log.Fatal(err)
	}

	a.fetchUsers(ctx)

	ctx.ObserveState("entity", &a.entity).
		OnChange(func() {
			log.Println("a.entity: ", a.entity)
		})

	ctx.ObserveState("termsAccepted", &a.termsAccepted).
		OnChange(func() {
			log.Println("a.termsAccepted: ", a.termsAccepted)
			log.Println("a.termsAccepted.entity: ", a.entity)

			if a.entity == "individual" {
				a.beginRegistration(ctx)
			}
		})

	ctx.ObserveState("vat", &a.vat).
		OnChange(func() {
			ctx.GetState("businessName", &a.businessName)
			log.Println("a.vat.businessName: ", a.businessName)
			log.Println("a.vat: ", a.vat)
			log.Println("a.vat.entity: ", a.entity)
			log.Println("a.vat.termsAccepted: ", a.termsAccepted)

			if a.entity == "business" && a.termsAccepted {
				a.beginRegistration(ctx)
			}
		})
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
	app.Window().GetElementByID("main-menu").Call("click")
}

func (a *auth) doLogin(ctx app.Context, e app.Event) {
	descriptorJSON := e.Get("detail").Get("descriptor").String()

	if len(descriptorJSON) == 0 {
		log.Fatal("descriptorJSON is empty")
	}

	for _, user := range a.users {
		desc, err := json.Marshal(user.Descriptor)
		if err != nil {
			log.Fatal(err)
		}

		if string(desc) == descriptorJSON {
			ctx.SetState("userID", string(user.ID))
			if len(user.VAT) > 0 {
				ctx.SetState("isBusiness", true)
				ctx.SetState("businessName", user.Name)
			}
			a.beginLogin(ctx, string(user.CredentialIDs[0].ID))
		}
	}

}

func (a *auth) fetchUsers(ctx app.Context) {
	descriptors := [][]float32{}

	users := a.getUsers()

	a.users = users

	if len(users) > 0 {
		for _, user := range users {
			descriptors = append(descriptors, user.Descriptor)
		}
	}

	descriptorsJSON, err := json.Marshal(descriptors)
	if err != nil {
		log.Fatal(err)
	}

	// Send response event to child
	app.Window().Get("parent").Get("window").Call("dispatchEvent", // Target the iframe's window
		app.Window().Get("CustomEvent").New("descriptorsFetched", map[string]interface{}{
			"detail": map[string]interface{}{
				"descriptors": string(descriptorsJSON),
			},
		}),
	)

	days := daysRemainingInMonth(time.Now())
	if days <= 3 {
		a.getIncome(ctx)
	}
}

func (a *auth) getUser(key, value string) User {
	u, err := a.sh.OrbitDocsQueryEnc(dbUser, key, value)
	if err != nil {
		log.Println("Error querying for user:", err)
		return User{}
	}

	// Directly unmarshal the byte slice into the User struct
	users := []User{}
	err = json.Unmarshal(u, &users) // Unmarshal the byte slice directly
	if err != nil {
		log.Println("Error unmarshaling user data:", err)
		return User{}
	}

	return users[0]
}

func (a *auth) getUsers() []*User {
	res, err := a.sh.OrbitDocsQueryEnc(dbUser, "all", "")
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

	users := []*User{}

	if len(res) == 0 {
		return users
	}

	err = json.Unmarshal([]byte(res7), &users)
	if err != nil {
		log.Fatal(err)
	}

	return users
}

func (a *auth) deleteUsers() {
	err := a.sh.OrbitDocsDelete(dbUser, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) createUser(ctx app.Context, userID, credentialID string) {
	ctx.Async(func() {
		var descriptorBytes []float32
		err := json.Unmarshal([]byte(a.descriptorJSON), &descriptorBytes)
		if err != nil {
			log.Fatal(err)
		}

		user := &User{
			Name: a.businessName,
			ID:   protocol.URLEncodedBase64(userID),
			CredentialIDs: []webauthn.Credential{
				{
					ID: []byte(credentialID),
				},
			},
			Descriptor: descriptorBytes,
			VAT:        a.vat,
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
			a.user = user
		})
	})
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
				ctx.SetState("businessName", a.businessName)
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
