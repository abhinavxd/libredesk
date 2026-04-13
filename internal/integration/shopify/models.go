package shopify

import "time"

// CustomerResult is the top-level response from a customer lookup.
type CustomerResult struct {
	Found     bool       `json:"found"`
	Customer  Customer   `json:"customer,omitempty"`
	Orders    []Order    `json:"orders,omitempty"`
	FetchedAt time.Time  `json:"fetched_at"`
}

// Customer holds Shopify customer profile data.
type Customer struct {
	ID             string   `json:"id"`
	FirstName      string   `json:"first_name"`
	LastName       string   `json:"last_name"`
	Email          string   `json:"email"`
	Phone          string   `json:"phone"`
	CreatedAt      string   `json:"created_at"`
	NumberOfOrders string   `json:"number_of_orders"`
	TotalSpent     string   `json:"total_spent"`
	CurrencyCode   string   `json:"currency_code"`
	DefaultAddress *Address `json:"default_address,omitempty"`
}

// Address is a Shopify mailing address.
type Address struct {
	City         string `json:"city"`
	Province     string `json:"province"`
	ProvinceCode string `json:"province_code"`
	Country      string `json:"country"`
	CountryCode  string `json:"country_code"`
	Zip          string `json:"zip"`
}

// Order holds Shopify order data.
type Order struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	CreatedAt         string     `json:"created_at"`
	FinancialStatus   string     `json:"financial_status"`
	FulfillmentStatus string     `json:"fulfillment_status"`
	TotalPrice        string     `json:"total_price"`
	CurrencyCode      string     `json:"currency_code"`
	LineItems         []LineItem `json:"line_items"`
}

// LineItem is one product line inside an order.
type LineItem struct {
	Title    string `json:"title"`
	Quantity int    `json:"quantity"`
	Price    string `json:"price"`
	ImageURL string `json:"image_url,omitempty"`
}

// -- Raw GraphQL response structs (internal) --

type graphQLError struct {
	Message string `json:"message"`
}

type customerNode struct {
	ID             string          `json:"id"`
	FirstName      string          `json:"firstName"`
	LastName       string          `json:"lastName"`
	Email          string          `json:"email"`
	Phone          string          `json:"phone"`
	CreatedAt      string          `json:"createdAt"`
	NumberOfOrders string          `json:"numberOfOrders"`
	AmountSpent    moneyV2         `json:"amountSpent"`
	DefaultAddress *addressNode    `json:"defaultAddress"`
	Orders         orderEdges      `json:"orders"`
}

type moneyV2 struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}

type addressNode struct {
	City           string `json:"city"`
	Province       string `json:"province"`
	ProvinceCode   string `json:"provinceCode"`
	Country        string `json:"country"`
	CountryCodeV2  string `json:"countryCodeV2"`
	Zip            string `json:"zip"`
}

type orderEdges struct {
	Edges []struct {
		Node orderNode `json:"node"`
	} `json:"edges"`
}

type orderNode struct {
	ID                       string        `json:"id"`
	Name                     string        `json:"name"`
	CreatedAt                string        `json:"createdAt"`
	DisplayFinancialStatus   string        `json:"displayFinancialStatus"`
	DisplayFulfillmentStatus string        `json:"displayFulfillmentStatus"`
	TotalPriceSet            totalPriceSet `json:"totalPriceSet"`
	LineItems                lineItemEdges `json:"lineItems"`
}

type totalPriceSet struct {
	ShopMoney moneyV2 `json:"shopMoney"`
}

type lineItemEdges struct {
	Edges []struct {
		Node lineItemNode `json:"node"`
	} `json:"edges"`
}

type lineItemNode struct {
	Title                string          `json:"title"`
	Quantity             int             `json:"quantity"`
	OriginalUnitPriceSet *totalPriceSet  `json:"originalUnitPriceSet"`
	Image                *imageNode      `json:"image"`
}

type imageNode struct {
	URL string `json:"url"`
}
