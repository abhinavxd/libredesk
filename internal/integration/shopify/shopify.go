// Package shopify provides a client for the Shopify Admin GraphQL API,
// focused on customer lookups for the conversation sidebar.
package shopify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/zerodha/logf"
)

const (
	graphqlAPIVersion = "2025-01"
	cacheTTL          = 5 * time.Minute
)

// Client talks to the Shopify Admin GraphQL API.
type Client struct {
	httpClient *http.Client
	lo         *logf.Logger

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	data      *CustomerResult
	expiresAt time.Time
}

// New creates a new Shopify client.
func New(lo *logf.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		lo:         lo,
		cache:      make(map[string]cacheEntry),
	}
}

// LookupCustomer searches Shopify for a customer by email.
func (c *Client) LookupCustomer(storeURL, accessToken, email string) (*CustomerResult, error) {
	cacheKey := storeURL + ":" + email

	c.mu.RLock()
	if entry, ok := c.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.data, nil
	}
	c.mu.RUnlock()

	result, err := c.fetchCustomer(storeURL, accessToken, email)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[cacheKey] = cacheEntry{data: result, expiresAt: time.Now().Add(cacheTTL)}
	c.mu.Unlock()

	return result, nil
}

// ExchangeOAuthToken exchanges a temporary OAuth code for a permanent access token.
func (c *Client) ExchangeOAuthToken(storeURL, apiKey, apiSecret, code string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"client_id":     apiKey,
		"client_secret": apiSecret,
		"code":          code,
	})

	url := fmt.Sprintf("https://%s/admin/oauth/access_token", storeURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Shopify OAuth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading OAuth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Shopify OAuth returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing OAuth response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("Shopify returned empty access token")
	}

	return result.AccessToken, nil
}

// TestConnection verifies the credentials by fetching the shop name.
func (c *Client) TestConnection(storeURL, accessToken string) (string, error) {
	query := `{ shop { name myshopifyDomain } }`
	resp, err := c.doGraphQL(storeURL, accessToken, query, nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Data struct {
			Shop struct {
				Name             string `json:"name"`
				MyshopifyDomain  string `json:"myshopifyDomain"`
			} `json:"shop"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("error parsing Shopify response: %w", err)
	}
	if len(result.Errors) > 0 {
		return "", fmt.Errorf("Shopify API error: %s", result.Errors[0].Message)
	}
	return result.Data.Shop.Name, nil
}

func (c *Client) fetchCustomer(storeURL, accessToken, email string) (*CustomerResult, error) {
	query := `
	query customerByEmail($query: String!) {
		customers(first: 1, query: $query) {
			edges {
				node {
					id
					firstName
					lastName
					email
					phone
					createdAt
					numberOfOrders
					amountSpent {
						amount
						currencyCode
					}
					defaultAddress {
						city
						province
						provinceCode
						country
						countryCodeV2
						zip
					}
					orders(first: 5, sortKey: CREATED_AT, reverse: true) {
						edges {
							node {
								id
								name
								createdAt
								displayFinancialStatus
								displayFulfillmentStatus
								totalPriceSet {
									shopMoney {
										amount
										currencyCode
									}
								}
								lineItems(first: 10) {
									edges {
										node {
											title
											quantity
											originalUnitPriceSet {
												shopMoney {
													amount
													currencyCode
												}
											}
											image {
												url
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	vars := map[string]interface{}{
		"query": fmt.Sprintf("email:%s", email),
	}

	resp, err := c.doGraphQL(storeURL, accessToken, query, vars)
	if err != nil {
		return nil, err
	}

	var gqlResp struct {
		Data struct {
			Customers struct {
				Edges []struct {
					Node customerNode `json:"node"`
				} `json:"edges"`
			} `json:"customers"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}
	if err := json.Unmarshal(resp, &gqlResp); err != nil {
		return nil, fmt.Errorf("error parsing Shopify response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("Shopify API error: %s", gqlResp.Errors[0].Message)
	}

	if len(gqlResp.Data.Customers.Edges) == 0 {
		return &CustomerResult{Found: false}, nil
	}

	node := gqlResp.Data.Customers.Edges[0].Node
	result := &CustomerResult{
		Found:     true,
		Customer:  mapCustomer(node),
		Orders:    mapOrders(node.Orders),
		FetchedAt: time.Now(),
	}
	return result, nil
}

func (c *Client) doGraphQL(storeURL, accessToken, query string, variables map[string]interface{}) ([]byte, error) {
	body := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		body["variables"] = variables
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s/admin/api/%s/graphql.json", storeURL, graphqlAPIVersion)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Shopify-Access-Token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Shopify request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Shopify response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Shopify API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func mapCustomer(n customerNode) Customer {
	var addr *Address
	if n.DefaultAddress != nil {
		addr = &Address{
			City:         n.DefaultAddress.City,
			Province:     n.DefaultAddress.Province,
			ProvinceCode: n.DefaultAddress.ProvinceCode,
			Country:      n.DefaultAddress.Country,
			CountryCode:  n.DefaultAddress.CountryCodeV2,
			Zip:          n.DefaultAddress.Zip,
		}
	}
	return Customer{
		ID:              n.ID,
		FirstName:       n.FirstName,
		LastName:        n.LastName,
		Email:           n.Email,
		Phone:           n.Phone,
		CreatedAt:       n.CreatedAt,
		NumberOfOrders:  n.NumberOfOrders,
		TotalSpent:      n.AmountSpent.Amount,
		CurrencyCode:    n.AmountSpent.CurrencyCode,
		DefaultAddress:  addr,
	}
}

func mapOrders(edges orderEdges) []Order {
	out := make([]Order, 0, len(edges.Edges))
	for _, e := range edges.Edges {
		o := e.Node
		order := Order{
			ID:                o.ID,
			Name:              o.Name,
			CreatedAt:         o.CreatedAt,
			FinancialStatus:   o.DisplayFinancialStatus,
			FulfillmentStatus: o.DisplayFulfillmentStatus,
			TotalPrice:        o.TotalPriceSet.ShopMoney.Amount,
			CurrencyCode:      o.TotalPriceSet.ShopMoney.CurrencyCode,
			LineItems:         make([]LineItem, 0, len(o.LineItems.Edges)),
		}
		for _, li := range o.LineItems.Edges {
			item := LineItem{
				Title:    li.Node.Title,
				Quantity: li.Node.Quantity,
			}
			if li.Node.OriginalUnitPriceSet != nil {
				item.Price = li.Node.OriginalUnitPriceSet.ShopMoney.Amount
			}
			if li.Node.Image != nil {
				item.ImageURL = li.Node.Image.URL
			}
			order.LineItems = append(order.LineItems, item)
		}
		out = append(out, order)
	}
	return out
}
