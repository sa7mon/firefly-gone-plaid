package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/plaid/plaid-go/plaid"
	"io/ioutil"
	"net/http"
)

type Config struct {
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

func (c Connection) GetAccountByPlaidAccountId(plaidId string) (Account, error){
	for _, account := range c.Accounts {
		if account.PlaidAccountId == plaidId {
			return account, nil
		}
	}
	return Account{}, errors.New("couldn't find account")
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
					connAccount.PlaidAccountId = respAccount.AccountID
					fmt.Println("Matched Plaid account")
					matchedAccount = true
					break
				}
			}
			if matchedAccount == false {
				fmt.Println("Warning: Couldn't match Plaid account id", respAccount.AccountID, "to an account in the config")
			}
		}
	}

}
