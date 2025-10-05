package xifer

import (
	"errors"
	"strings"

	"github.com/ciphermountain/exchange-client/pkg/restclient"
)

type Symbol string

func ParseSymbolString(input string) (restclient.SymbolType, error) {
	switch input {
	case string(restclient.BTC):
		return restclient.BTC, nil
	case string(restclient.ETH):
		return restclient.ETH, nil
	case string(restclient.USDT):
		return restclient.USDT, nil
	case string(restclient.XIFR):
		return restclient.XIFR, nil
	default:
		return "", errors.New("invalid symbol type")
	}
}

type Market string

func ParseMarketString(input string) (Market, error) {
	symbols := strings.Split(input, "-")
	if len(symbols) != 2 {
		return "", errors.New("invalid market string")
	}

	base, err := ParseSymbolString(symbols[0])
	if err != nil {
		return "", err
	}

	quote, err := ParseSymbolString(symbols[1])
	if err != nil {
		return "", err
	}

	return Market(strings.Join([]string{string(base), string(quote)}, "-")), nil
}
