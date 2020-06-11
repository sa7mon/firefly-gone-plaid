# Firefly Gone Plaid

A connector to add financial transactions from the Plaid API to [Firefly III](https://github.com/firefly-iii/firefly-iii)

<p align="center">
<img width="480" height="260" src="https://user-images.githubusercontent.com/3712226/84401910-7355cb00-abc9-11ea-92f4-3be6fa398f7a.png"><br />"Firefly.....it's gone Plaid!"
</p>

## Usage

**Standalone**

```
go build -o firefly-gone-plaid
./firefly-gone-plaid --config config.json --start-date 2020-06-10 --end-date 2020-06-20
```

**Docker**

```
docker pull hothamandcheese/firefly-gone-plaid:latest
docker run --rm -v $(pwd)/config:/config:ro hothamandcheese/firefly-gone-plaid:latest --config /config/config.json --start-date 2020-05-15 --end-date 2020-06-11
```

## Config

Before running the tool, you'll need to:

1. Create Plaid account
2. Use the [Plaid Quickstart](https://github.com/plaid/quickstart) to connect your bank accounts to your Plaid account and get your Plaid `access-development-X` tokens
3. Create a `config.json` file that follows this basic schema:

```json
{
  "firefly_api_base_url": "http://ip.or.hostname.of.firefly:port",
  "firefly_token": "XXXXXXXXXXX",
  "plaid_client_id": "XXXXXXXXXX",
  "plaid_secret": "XXXXXXXXXXXXXX",
  "plaid_public_key": "XXXXXXXXXXXXXX",
  "connections": [
    {
      "token": "access-development-XXXXX-XXXXXXXX",
      "institution_nickname": "US Bank",
      "accounts": [
        {
          "firefly_account_id": 1,
          "account_last_four": "1111",
          "account_nickname": "US Bank Savings"
        },
        {
          "firefly_account_id": 3,
          "account_last_four": "2222",
          "account_nickname": "US Bank Checking"
        }
      ]
    },
    {
      "token": "access-development-XXXXXXX-XXXXXXXXX",
      "institution_nickname": "Discover",
      "accounts": [
        {
          "firefly_account_id": 7,
          "account_last_four": "4567",
          "account_nickname": "Discover"
        }
      ]
    }
  ]
}
```

**Note:**
* All "*_nickname" fields can be set to anything. The names will only be used in logs.
* The `firefly_account_id` must be an ID of a Firefly asset account. Expense and revenue accounts do not currently work.
