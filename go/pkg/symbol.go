package xifer

import (
	"errors"
	"strings"

	"github.com/ciphermountain/exchange-client/go/pkg/rest"
)

type Symbol string

func ParseSymbolString(input string) (rest.SymbolType, error) {
	switch input {
	case string(rest.BTC):
		return rest.BTC, nil
	case string(rest.ETH):
		return rest.ETH, nil
	case string(rest.USDT):
		return rest.USDT, nil
	case string(rest.XIFR):
		return rest.XIFR, nil
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
