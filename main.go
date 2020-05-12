package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/plaid/plaid-go/plaid"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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
	ErrorIfDuplicateHash bool   `json:"error_if_duplicate_hash"`
	ApplyRules bool 			`json:"apply_rules"`
	Transactions []Transaction  `json:"transactions"`
}

type Transaction struct {
	Type string `json:"type"`  					// deposit, withdrawal, transfer, reconciliation
	Date string `json:"date"`  					// YYYY-MM-DD
	Amount float64 `json:"amount"` 				// 11.11
	Description string `json:"description"`		// "Groceries"
	CurrencyId int `json:"currency_id"`			// 17 - USD
	CategoryName string `json:"category_name"`  // "Food" - doesn't have to exist in Firefly already
	SourceID int `json:"source_id"`				// ID of Firefly account for withdrawal
	DestinationID int `json:"destination_id"`	// ID of Firefly account for deposit
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
	if ptrans.Date == "" {
		return Transaction{}, errors.New("required field 'Date' is blank")
	}
	t := Transaction{}
	if ptrans.Amount < 0 {
		t.Type = "deposit"
		t.Amount = ptrans.Amount * -1
		t.DestinationID = fireflyAccountId
	} else {
		t.Type = "withdrawal"
		t.Amount = ptrans.Amount
		t.SourceID = fireflyAccountId
	}
	t.Date = ptrans.Date
	t.Description = ptrans.Name
	t.CurrencyId = 17
	t.CategoryName = strings.Join(ptrans.Category, "|")
	t.Notes = "Imported by Firefly-Gone-Plaid"
	t.ExternalId = ptrans.ID
	return t, nil
}

func (t TransactionRequest) StoreTransaction(c Config) error {
	// Send transaction to Firefly API
	// POST http://bionic.home.lan:7080/api/v1/transactions
	// Content-Type: JSON
	// Headers:
	//		Authorization: Bearer Token
	//		Accept: application/json

	payload, err := json.Marshal(t)
	if err != nil {
		return errors.New("Error: " + err.Error())
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/transactions", c.FireflyApiBaseUrl),
		bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Add("User-Agent", "Firefly-Gone-Plaid v0.1")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer " + c.FireflyToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fireflyError := fmt.Sprintf("%q - %q\n%+v", resp.Status, bodyString, t)
		return errors.New(fireflyError)
	}
	return nil
}

func main() {
	configFilePath := flag.String("config", "", "Path to config file. (Required)")
	startDate := flag.String("start-date", "", "Start date of range to fetch transactions for. (Required)")
	endDate := flag.String("end-date", "", "End date of range to fetch transactions for. (Required)")
	flag.Parse()

	if *configFilePath == ""{
		fmt.Printf("Error: config file path required.")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *startDate == ""{
		fmt.Printf("Error: Start Date required.")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *endDate == ""{
		fmt.Printf("Error: End Date required.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// read file
	data, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		fmt.Print(err)
	}

	// Read JSON file in
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Println("Error reading JSON file: ", err)
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

	transactionOptions := plaid.GetTransactionsOptions{
		StartDate: *startDate,
		EndDate: *endDate,
		Count: 500,
	}
	for _, connection := range config.Connections {
		resp, err := client.GetTransactionsWithOptions(connection.Token, transactionOptions)
		if err != nil {
			log.Println("Error getting transactions: ", err)
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

		log.Println(fmt.Sprintf("Got %d Plaid transactions to process",len(resp.Transactions)))

		for i, plaidTransaction := range resp.Transactions {
			id, a := plaid2fireflyId[plaidTransaction.AccountID]
			if !a {
				log.Println("Warning: unknown account ID:", id)
				continue
			}
			if plaidTransaction.Pending {
				log.Println("Warning: Skipping pending transaction: ", plaidTransaction.ID)
				continue
			}

			t, terr := MakeTransaction(plaidTransaction, id)
			if terr != nil {
				fmt.Println("Error creating transaction: ", terr)
			}
			transactions := make([]Transaction, 0) // TODO: Simplify this
			transactions = append(transactions, t)

			fireflyTransaction := TransactionRequest{
				ErrorIfDuplicateHash: true,
				ApplyRules: true,
				Transactions: transactions,
			}
			err = fireflyTransaction.StoreTransaction(config) // Send to Firefly API
			if err != nil {
				if strings.Contains(err.Error(), "Duplicate of transaction #") {
					log.Println(fmt.Sprintf("Transaction %d: duplicate", i))
				} else {
					log.Println(fmt.Sprintf("Transaction %d: error", i))
					log.Println(err.Error())
				}
			} else {
				log.Println(fmt.Sprintf("Transaction %d: processed", i))
			}
		}
	}
}
