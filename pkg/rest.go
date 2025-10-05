package xifer

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	client "github.com/ciphermountain/exchange-client/pkg/restclient"
)

var (
	ErrValidation = errors.New("validation error")
)

type Order struct {
	ID string
}

type ClientOption func(*client.Client) error

type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *client.Client) error {
		c.Client = doer
		return nil
	}
}

type RestClient struct {
	url    string
	client *client.Client
}

func NewRestClient(url, token string, opts ...ClientOption) (*RestClient, error) {
	restOpts := []client.ClientOption{
		client.WithRequestEditorFn(withXiferToken(token)),
	}

	for _, opt := range opts {
		restOpts = append(restOpts, client.ClientOption(opt))
	}

	clnt, err := client.NewClient(url, restOpts...)
	if err != nil {
		return nil, err
	}

	return &RestClient{
		url:    url,
		client: clnt,
	}, nil
}

func (c *RestClient) GetMarketSnapshot(ctx context.Context, market Market, asks, bids bool) (client.SnapshotItemList, error) {
	if !asks && !bids {
		return nil, errors.New("select either asks or bids or both to return data")
	}

	params := &client.GetV1MarketsMarketSnapshotParams{
		Asks: &asks,
		Bids: &bids,
	}

	rawResp, err := c.client.GetV1MarketsMarketSnapshot(ctx, client.MarketParam(market), params)
	if err != nil {
		return nil, err
	}

	resp, err := client.ParseGetV1MarketsMarketSnapshotResponse(rawResp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode(), resp.Status())
	}

	list := resp.JSON200.Data

	if list == nil {
		return nil, fmt.Errorf("unexpected list output data")
	}

	return *list, nil
}

func (c *RestClient) CreateOrder(
	ctx context.Context,
	accountID string,
	request client.OrderRequest,
) (client.BookOrder, error) {
	resp, err := c.client.PostV1AccountsAccountIDOrders(ctx, accountID, request)
	if err != nil {
		return client.BookOrder{}, fmt.Errorf("%w: order post failure: %w", ErrConnection, err)
	}

	odr, err := client.ParsePostV1AccountsAccountIDOrdersResponse(resp)
	if err != nil {
		return client.BookOrder{}, fmt.Errorf("%w: failed to parse order post response: %w", ErrEncoding, err)
	}

	switch odr.StatusCode() {
	case http.StatusConflict:
		if odr.JSON409.Errors != nil {
			if errs := *odr.JSON409.Errors; len(errs) > 0 {
				return client.BookOrder{}, fmt.Errorf("%w: post order failure: %s", ErrValidation, errs[0].Detail)
			}
		}

		return client.BookOrder{}, fmt.Errorf("api returned with code %d; %s", odr.StatusCode(), odr.Status())
	case http.StatusInternalServerError:
		if odr.JSON500.Errors != nil {
			if errs := *odr.JSON500.Errors; len(errs) > 0 {
				return client.BookOrder{}, fmt.Errorf("%w: post order failure: %s", ErrValidation, errs[0].Detail)
			}
		}

		return client.BookOrder{}, fmt.Errorf("%w: api returned with code %d; %s", ErrValidation, odr.StatusCode(), odr.Status())
	}

	return *odr.JSON200.Data, nil
}

func (c *RestClient) GetOrderDetail(ctx context.Context, accountID, orderID string) (client.BookOrder, error) {
	resp, err := c.client.GetV1AccountsAccountIDOrdersOrderID(ctx, accountID, orderID)
	if err != nil {
		return client.BookOrder{}, err
	}

	data, err := client.ParseGetV1AccountsAccountIDOrdersOrderIDResponse(resp)
	if err != nil {
		return client.BookOrder{}, err
	}

	if data.StatusCode() != http.StatusOK {
		return client.BookOrder{}, fmt.Errorf("unexpected status: %s", data.Status())
	}

	return *data.JSON200.Data, nil
}

func (c *RestClient) GetTransaction(ctx context.Context, accountID, transactionID string) (client.Transaction, error) {
	trxn := client.Transaction{}

	resp, err := c.client.GetV1AccountsAccountIDTransactionsTransactionID(ctx, accountID, transactionID)
	if err != nil {
		return trxn, err
	}

	data, err := client.ParseGetV1AccountsAccountIDTransactionsTransactionIDResponse(resp)
	if err != nil {
		return trxn, err
	}

	if data.StatusCode() != http.StatusOK {
		return trxn, fmt.Errorf("unexpected status: %s", data.Status())
	}

	return *data.JSON200.Data, nil
}

func (c *RestClient) GetAccounts(ctx context.Context) ([]client.Account, error) {
	resp, err := c.client.GetV1Accounts(ctx)
	if err != nil {
		return nil, err
	}

	data, err := client.ParseGetV1AccountsResponse(resp)
	if err != nil {
		return nil, err
	}

	if data.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", data.Status())
	}

	return *data.JSON200.Data, nil
}

func (c *RestClient) GetAccount(ctx context.Context, accountID string) (client.Account, error) {
	resp, err := c.client.GetV1AccountsAccountID(ctx, accountID)
	if err != nil {
		return client.Account{}, err
	}

	data, err := client.ParseGetV1AccountsAccountIDResponse(resp)
	if err != nil {
		return client.Account{}, err
	}

	if data.StatusCode() != http.StatusOK {
		return client.Account{}, fmt.Errorf("unexpected status: %d %s", data.StatusCode(), data.Status())
	}

	return *data.JSON200.Data, nil
}

func (c *RestClient) GetAddress(ctx context.Context, symbol string, accountID *string) (string, error) {
	if accountID == nil {
		accounts, err := c.GetAccounts(ctx)
		if err != nil {
			return "", err
		}

		if len(accounts) == 0 {
			return "", fmt.Errorf("expected more than one account")
		}

		accountID = &accounts[0].Id
	}

	resp, err := c.client.GetV1AccountsAccountIDAddressesSymbolName(ctx, *accountID, client.SymbolType(symbol))
	if err != nil {
		return "", err
	}

	data, err := client.ParseGetV1AccountsAccountIDAddressesSymbolNameResponse(resp)
	if err != nil {
		return "", fmt.Errorf("GetV1Addresses: %s", err)
	}

	if data.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d %s", data.StatusCode(), data.Status())
	}

	return data.JSON200.Data.Address, nil
}

func withXiferToken(token string) client.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Add("Authorization", fmt.Sprintf("APIKEY:%s", token))

		return nil
	}
}
