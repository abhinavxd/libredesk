// Test server for AI assistant custom tools. Point tool URLs at
// http://localhost:7070/<endpoint> and set auth header X-Api-Key: test-secret-token.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
)

const (
	apiKeyHeader = "X-Api-Key"
	apiKey       = "test-secret-token"

	contactIDHeader       = "X-Libredesk-Contact-External-Id"
	contactEmailHeader    = "X-Libredesk-Contact-Email"
	contactVerifiedHeader = "X-Libredesk-Contact-Verified"
)

// users maps the contact's external user id (as set on the libredesk contact)
// to their account data. Edit this to match your test contacts.
var users = map[string]User{
	"USR1001": {
		Email:   "alice@example.com",
		Name:    "Alice",
		Plan:    "pro",
		Balance: 2450.75,
		KYC:     "verified",
		Orders: []Order{
			{ID: "ORD-9001", Item: "Nifty 50 ETF", Qty: 10, Status: "executed"},
			{ID: "ORD-9002", Item: "Gold Bees", Qty: 25, Status: "pending"},
		},
	},
	"USR1002": {
		Email:   "bob@example.com",
		Name:    "Bob",
		Plan:    "free",
		Balance: 0,
		KYC:     "pending",
		Orders:  []Order{},
	},
}

type Order struct {
	ID     string `json:"id"`
	Item   string `json:"item"`
	Qty    int    `json:"qty"`
	Status string `json:"status"`
}

type User struct {
	Email   string  `json:"email"`
	Name    string  `json:"name"`
	Plan    string  `json:"plan"`
	Balance float64 `json:"balance"`
	KYC     string  `json:"kyc"`
	Orders  []Order `json:"orders"`
}

func main() {
	addr := flag.String("addr", ":7070", "listen address")
	flag.Parse()

	http.HandleFunc("/account", withUser(handleAccount))
	http.HandleFunc("/orders", withUser(handleOrders))
	http.HandleFunc("/balance", withUser(handleBalance))

	log.Printf("tools server listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handleAccount(w http.ResponseWriter, r *http.Request, u User) {
	writeJSON(w, map[string]any{
		"name": u.Name, "email": u.Email, "plan": u.Plan, "kyc": u.KYC,
	})
}

func handleOrders(w http.ResponseWriter, r *http.Request, u User) {
	writeJSON(w, map[string]any{"orders": u.Orders})
}

func handleBalance(w http.ResponseWriter, r *http.Request, u User) {
	writeJSON(w, map[string]any{"balance": u.Balance, "currency": "INR"})
}

// withUser validates the auth header and the verified flag, resolves the contact's
// external id to a user, and cross-checks the contact email against the mapped account.
// Identity is read only from the headers libredesk injects, never from the model's args.
func withUser(next func(http.ResponseWriter, *http.Request, User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(contactIDHeader)
		email := r.Header.Get(contactEmailHeader)
		verified := r.Header.Get(contactVerifiedHeader)
		body, _ := io.ReadAll(r.Body)
		log.Printf("%s %s external_id=%q email=%q verified=%q args=%s", r.Method, r.URL.Path, id, email, verified, body)

		if r.Header.Get(apiKeyHeader) != apiKey {
			writeError(w, http.StatusUnauthorized, "invalid or missing "+apiKeyHeader)
			return
		}
		if verified != "true" {
			writeError(w, http.StatusForbidden, "contact is not verified; a self-claimed email is not enough for account data")
			return
		}
		if id == "" {
			writeError(w, http.StatusBadRequest, "no contact external id on request; set an external user id on the contact")
			return
		}
		u, ok := users[id]
		if !ok {
			writeError(w, http.StatusNotFound, "no user found for external id "+id)
			return
		}
		if email != "" && email != u.Email {
			writeError(w, http.StatusForbidden, "contact email does not match the account on file for "+id)
			return
		}
		next(w, r, u)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	out, _ := json.Marshal(v)
	log.Printf("-> 200 %s", out)
	w.Write(out)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	log.Printf("-> %d %s", code, msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
