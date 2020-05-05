package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/plaid/plaid-go/plaid"
	"io/ioutil"
	"net/http"
	"strings"
)

type Config struct {
	FireflyApiBaseUrl string `json:"firefly_api_base_url"`
	FireflyToken   string `json:"firefly_token"`
	PlaidClientId  string `json:"plaid_client_id"`
	PlaidSecret    string `json:"plaid_secret"`
	PlaidPublicKey string `json:"plaid_public_key"`
	Connections    []Connection `json:"connections"`
}

type Connection struct {
	Token string `json:"token"`
	InstitutionNickname string `json:"institution_nickname"`
	Accounts []Account `json:"accounts"`
	//GetAccountByPlaidAccountId() (Account, error)
}

type Account struct {
	FireflyAccountId int `json:"firefly_account_id"`
	AccountLastFour string `json:"account_last_four"`
	AccountNickname string `json:"account_nickname"`
	PlaidAccountId string
}

type TransactionRequest struct {
	ErrorIfDuplicateHash bool `json:"error_if_duplicate_hash"`
	ApplyRules bool `json:"apply_rules"`
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	Type string `json:"type"`  					// deposit, withdrawl, transfer, reconciliation
	Date string `json:"date"`  					// YYYY-MM-DD
	Amount float64 `json:"amount"` 				// 11.11
	Description string `json:"description"`		// "Groceries"
	CurrencyId int `json:"currency_id"`			// 17 - USD
	CategoryName string `json:"category_name"`  // "Food" - doesn't have to exist in Firefly already
	SourceID int `json:"source_id"`				// ID of Firefly account
	Notes string `json:"notes"`					// "Imported by Firefly-Gone-Plaid"
	ExternalId string `json:"external_id"`		// Plaid transaction ID?
}

func (c Connection) GetAccountByPlaidAccountId(plaidId string) (Account, error){
	for _, account := range c.Accounts {
		fmt.Println(account.PlaidAccountId)
		if account.PlaidAccountId == plaidId {
			return account, nil
		}
	}
	return Account{}, errors.New("couldn't find account")
}

func MakeTransaction(ptrans plaid.Transaction, fireflyAccountId int) (Transaction, error) {
	t := Transaction{}
	if ptrans.Amount < 0 {
		t.Type = "deposit"
	} else {
		t.Type = "withdrawl"
	}
	t.Date = ptrans.Date
	t.Amount = ptrans.Amount
	t.Description = ptrans.Name
	t.CurrencyId = 17
	t.CategoryName = strings.Join(ptrans.Category, "|")
	t.SourceID = fireflyAccountId
	t.Notes = "Imported by Firefly-Gone-Plaid"
	t.ExternalId = ptrans.ID
	return t, nil
}

func (t TransactionRequest) StoreTransaction(c Config) {
	// Send transaction to Firefly API
	// POST http://bionic.home.lan:7080/api/v1/transactions
	// Content-Type: JSON
	// Headers:
	//		Authorization: Bearer Token
	//		Accept: application/json


}

func main() {
	// read file
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Print(err)
	}

	// Read JSON file in
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error reading JSON file: ", err)
	}

	clientOptions := plaid.ClientOptions{
		ClientID:    config.PlaidClientId,
		Secret:      config.PlaidSecret,
		PublicKey:   config.PlaidPublicKey,
		Environment: plaid.Development,    // Available environments are Sandbox, Development, and Production
		HTTPClient:  &http.Client{}, 	   // This parameter is optional
	}
	client, err := plaid.NewClient(clientOptions)

	plaid2fireflyId := make(map[string]int)

	for _, connection := range config.Connections {
		resp, err := client.GetTransactions(connection.Token, "2020-04-01", "2020-05-01")
		if err != nil {
			fmt.Println("Error getting transactions: ", err)
		}

		for _, respAccount := range resp.Accounts {
			// for each account listed in the response, match it with an account from the JSON config file
			matchedAccount := false
			for _, connAccount := range connection.Accounts {
				if respAccount.Mask == connAccount.AccountLastFour {
					plaid2fireflyId[respAccount.AccountID] = connAccount.FireflyAccountId
					matchedAccount = true
					break
				}
			}
			if matchedAccount == false {
				fmt.Println("Warning: Couldn't match Plaid account id", respAccount.AccountID, "to an account in the config")
			}
		}

		for _, plaidTransaction := range resp.Transactions {
			id, a := plaid2fireflyId[plaidTransaction.AccountID]
			if !a {
				fmt.Println("Warning: unknown account ID:", id)
				continue
			}
			if plaidTransaction.Pending {
				fmt.Println("Warning: Skipping pending transaction: ", plaidTransaction.ID)
				continue
			}

			t, terr := MakeTransaction(plaidTransaction, id)
			if terr != nil {
				fmt.Println("Error creating transaction: ", terr)
			}
			transactions := make([]Transaction, 1) // TODO: Simplify this
			transactions = append(transactions, t)

			fireflyTransaction := TransactionRequest{
				ErrorIfDuplicateHash: true,
				ApplyRules: true,
				Transactions: transactions,
			}
			fireflyTransaction.StoreTransaction(config) // Send to Firefly API
		}

	}
}
