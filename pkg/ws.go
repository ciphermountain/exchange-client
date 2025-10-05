package xifer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"github.com/ciphermountain/exchange-client/pkg/messages"
)

var (
	ErrWebsocketClientStopped = errors.New("websocket client stopped")
	ErrConnection             = errors.New("connection error")
	ErrEncoding               = errors.New("encoding error")
	ErrClientRunning          = errors.New("websocket client already running")
)

type EventType string

const (
	Open      EventType = "OPEN"
	Partial   EventType = "PARTIAL"
	Filled    EventType = "FILLED"
	Cancelled EventType = "CANCELLED"
)

type SubscribeOpt func(context.Context, *WSClient)

func SubscribeHeartbeats() func(context.Context, *WSClient) {
	return func(ctx context.Context, client *WSClient) {
		if err := client.write(messageSubscribe{
			Type:    subscribe,
			Channel: heartbeat,
		}); err != nil {
			client.logger.ErrorContext(ctx, err.Error())
		}
	}
}

func SubscribeStatus(token, account string, orders, transactions bool) func(context.Context, *WSClient) {
	return func(ctx context.Context, client *WSClient) {
		if !orders && !transactions {
			return
		}

		events := []event{}
		if orders {
			events = append(events, order)
		}

		if transactions {
			events = append(events, transaction)
		}

		client.token = token
		client.account = account

		if err := client.write(messageSubscribe{
			Type:      subscribe,
			Channel:   user,
			Token:     token,
			AccountID: account,
			Events:    events,
		}); err != nil {
			client.logger.ErrorContext(ctx, err.Error())
		}
	}
}

func SubscribeMarket(product string, changes, trades, asks, bids bool) func(context.Context, *WSClient) {
	return func(ctx context.Context, client *WSClient) {
		if _, err := ParseMarketString(product); err != nil {
			client.logger.ErrorContext(ctx, fmt.Sprintf("invalid product input: %s", product))
		}

		if !changes && !trades {
			return
		}

		if !asks && !bids {
			return
		}

		events := []event{}
		if changes {
			events = append(events, change)
		}

		if trades {
			events = append(events, trade)
		}

		sides := []side{}
		if asks {
			sides = append(sides, ask)
		}

		if bids {
			sides = append(sides, bid)
		}

		if err := client.write(messageSubscribe{
			Type:    subscribe,
			Channel: market,
			Events:  events,
			Product: product,
			Sides:   sides,
		}); err != nil {
			client.logger.ErrorContext(ctx, err.Error())
		}

	}
}

type WSClient struct {
	Errors       *MailBox[string]
	Heartbeats   *MailBox[messages.Heartbeat]
	Orders       *MailBox[messages.Order]
	Transactions *MailBox[messages.Transaction]
	Trades       *MailBox[messages.Trade]
	Changes      *MailBox[messages.Change]

	// dependencies and config
	uri     string
	token   string
	account string
	logger  *slog.Logger

	// event tracking
	chHeartbeats   chan eventHeartbeat
	chOrders       chan orderStatusMessage
	chTransactions chan transactionStatusMessage
	chTrades       chan eventMarketTrade
	chChanges      chan eventMarketChange

	// state variables
	conn *websocket.Conn

	// service state
	chStopSignal chan struct{}
	stopped      atomic.Bool
	stopping     atomic.Bool
}

func NewWSClient(uri string, logger *slog.Logger) *WSClient {
	return &WSClient{
		Errors:         newMailBox[string](5),
		Heartbeats:     newMailBox[messages.Heartbeat](5),
		Orders:         newMailBox[messages.Order](100),
		Transactions:   newMailBox[messages.Transaction](100),
		Trades:         newMailBox[messages.Trade](100),
		Changes:        newMailBox[messages.Change](100),
		uri:            uri,
		logger:         logger,
		chHeartbeats:   make(chan eventHeartbeat, 100),
		chOrders:       make(chan orderStatusMessage, 100),
		chTransactions: make(chan transactionStatusMessage, 100),
		chTrades:       make(chan eventMarketTrade, 100),
		chChanges:      make(chan eventMarketChange, 100),
	}
}

func (sc *WSClient) ConnectAndReadEvents(ctx context.Context, opts ...SubscribeOpt) error {
	defer sc.stopped.Store(true)

	sc.stopped.Store(false)
	sc.stopping.Store(false)
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	// TODO: retry on connection failure with exp. backoff

	dialer := websocket.DefaultDialer
	dialer.Subprotocols = []string{"json"}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	var h http.Header = map[string][]string{
		"User-Agent": {"Xifer WS Client"},
		"Accept":     {"*/*"},
		// "Accept-Encoding": {"gzip, deflate, br, zstd"},
		"Sec-Fetch-Dest": {"empty"},
		"Sec-Fetch-Mode": {"websocket"},
		"Sec-Fetch-Site": {"cross-site"},
		"Pragma":         {"no-cache"},
		"Cache-Control":  {"no-cache"},
	}

	conn, _, err := dialer.Dial(sc.uri, h)
	if err != nil {
		return fmt.Errorf("%w: failed to connect with dial error: %w", ErrConnection, err)
	}

	sc.conn = conn
	sc.conn.SetReadLimit(512)

	sc.chStopSignal = make(chan struct{})

	go sc.watchContext(ctx, cancel)
	go sc.watchHeartbeats(ctx)
	go sc.watchOrders(ctx)
	go sc.watchTransactions(ctx)
	go sc.watchTrades(ctx)
	go sc.watchChanges(ctx)

	for _, opt := range opts {
		opt(ctx, sc)
	}

	for {
		select {
		case <-sc.chStopSignal:
			return ErrWebsocketClientStopped
		default:
			_, rawMsg, err := sc.conn.ReadMessage()
			if err != nil {
				sc.logger.Debug(fmt.Sprintf("read error: %s", err))
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Printf("error: %v", err)
				}

				return nil
			}

			var msg message
			if err := json.Unmarshal(rawMsg, &msg); err != nil {
				return fmt.Errorf("%w: failed to unmarshal message: %w", ErrEncoding, err)
			}

			sc.extractEvents(msg)
		}
	}
}

func (sc *WSClient) watchContext(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	select {
	case <-sc.chStopSignal:
		return
	case <-ctx.Done():
		sc.Close()
	}
}

func (sc *WSClient) extractEvents(msg message) {
	if msg.Error != "" {
		sc.Errors.add(msg.Error)

		return
	}

	switch msg.Channel {
	case heartbeat:
		for _, rawEvent := range msg.Events {
			var event eventHeartbeat

			if err := json.Unmarshal(rawEvent, &event); err != nil {
				return
			}

			sc.chHeartbeats <- event
		}
	case user:
		for _, rawEvent := range msg.Events {
			var message userMessage

			if err := json.Unmarshal(rawEvent, &message); err != nil {
				return
			}

			if message.IsOrder() {
				sc.chOrders <- message.Order()

				continue
			}

			if message.IsTransaction() {
				sc.chTransactions <- message.Transaction()

				continue
			}
		}
	case market:
		for _, rawEvent := range msg.Events {
			var message eventMarket

			if err := json.Unmarshal(rawEvent, &message); err != nil {
				return
			}

			if message.IsTrade() {
				sc.chTrades <- message.Trade()

				continue
			}

			if message.IsChange() {
				sc.chChanges <- message.Change()

				continue
			}
		}
	}
}

func (sc *WSClient) watchHeartbeats(ctx context.Context) {
	for {
		select {
		case event := <-sc.chHeartbeats:
			sc.Heartbeats.add(messages.Heartbeat{
				CurrentTime: event.CurrentTime,
			})
		case <-ctx.Done():
			return
		}
	}
}

func (sc *WSClient) watchOrders(ctx context.Context) {
	for {
		select {
		case event := <-sc.chOrders:
			status, err := messages.ParseOrderStatus(event.Status)
			if err != nil {
				sc.logger.ErrorContext(ctx, "invalid order status")
			}

			sc.Orders.add(messages.Order{
				ID:      event.OrderID,
				Status:  status,
				Account: event.AccountID,
			})
		case <-ctx.Done():
			return
		}
	}
}

func (sc *WSClient) watchTransactions(ctx context.Context) {
	for {
		select {
		case event := <-sc.chTransactions:
			sc.Transactions.add(messages.Transaction{
				ID:      event.TransactionID,
				Status:  event.Status,
				Account: event.AccountID,
			})
		case <-ctx.Done():
			return
		}
	}
}

func (sc *WSClient) watchTrades(ctx context.Context) {
	for {
		select {
		case event := <-sc.chTrades:
			sc.Trades.add(messages.Trade{
				Market:    strings.Join([]string{event.Market.Base, event.Market.Quote}, "-"),
				Side:      event.Side,
				Price:     event.Price,
				ID:        event.ID,
				Timestamp: time.Time(event.Timestamp),
				Quantity:  event.Quantity,
			})
		case <-ctx.Done():
			return
		}
	}
}

func (sc *WSClient) watchChanges(ctx context.Context) {
	for {
		select {
		case event := <-sc.chChanges:
			sc.Changes.add(messages.Change{
				Market:    strings.Join([]string{event.Market.Base, event.Market.Quote}, "-"),
				Side:      event.Side,
				Price:     event.Price,
				Remaining: event.Remaining,
				Delta:     event.Delta,
				Action:    event.Action,
			})
		case <-ctx.Done():
			return
		}
	}
}

func (sc *WSClient) Close() {
	if sc.conn == nil || sc.stopped.Load() || sc.stopping.Load() {
		return
	}

	sc.stopping.Store(true)
	sc.conn.WriteMessage(websocket.CloseMessage, []byte{})

	_ = sc.conn.Close()
	close(sc.chStopSignal)
}

func (sc *WSClient) write(msg any) error {
	if sc.conn == nil {
		return fmt.Errorf("%w: connection closed", ErrConnection)
	}

	bMsg, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("%w: failed to encode message: %w", ErrEncoding, err)
	}

	log.Println(string(bMsg))

	if err := sc.conn.WriteMessage(websocket.TextMessage, bMsg); err != nil {
		return fmt.Errorf("%w: failed to send message for connection: %w", ErrConnection, err)
	}

	return nil
}

type messageAction string

const (
	subscribe messageAction = "subscribe"
	// unsubscribe messageAction = "unsubscribe"
)

type channel string

const (
	heartbeat channel = "heartbeat"
	user      channel = "user"
	market    channel = "market"
)

type event string

const (
	change      event = "change"
	trade       event = "trade"
	order       event = "order"
	transaction event = "transaction"
)

type side string

const (
	ask side = "ask"
	bid side = "bid"
)

type messageSubscribe struct {
	Type      messageAction `json:"type"`
	Channel   channel       `json:"channel"`
	Token     string        `json:"token,omitempty"`
	AccountID string        `json:"account_id,omitempty"`
	Events    []event       `json:"events,omitempty"`
	Product   string        `json:"product,omitempty"`
	Sides     []side        `json:"sides,omitempty"`
}

type message struct {
	Error   string            `json:"error"`
	Channel channel           `json:"channel"`
	Events  []json.RawMessage `json:"events"`
}

type eventHeartbeat struct {
	CurrentTime time.Time `json:"CurrentTime"`
}

type userMessage struct {
	statusMessage
	TransactionID string `json:"transaction_id"`
	OrderID       string `json:"order_id"`
}

func (m userMessage) IsTransaction() bool {
	return m.TransactionID != "" && m.OrderID == ""
}

func (m userMessage) Transaction() transactionStatusMessage {
	return transactionStatusMessage{
		statusMessage: m.statusMessage,
		TransactionID: m.TransactionID,
	}
}

func (m userMessage) IsOrder() bool {
	return m.TransactionID == "" && m.OrderID != ""
}

func (m userMessage) Order() orderStatusMessage {
	return orderStatusMessage{
		statusMessage: m.statusMessage,
		OrderID:       m.OrderID,
	}
}

type statusMessage struct {
	Status    string `json:"status"`
	AccountID string `json:"account_id"`
}

type transactionStatusMessage struct {
	statusMessage
	TransactionID string `json:"transaction_id"`
}

type orderStatusMessage struct {
	statusMessage
	OrderID string `json:"order_id"`
}

type eventMarket struct {
	eventMarketCommon
	ID        string          `json:"trade_id,omitempty"`
	Timestamp *xiferTime      `json:"timestamp,omitempty"`
	Quantity  decimal.Decimal `json:"decimal,omitempty"`
	Remaining decimal.Decimal `json:"remaining,omitempty"`
	Delta     decimal.Decimal `json:"delta,omitempty"`
	Action    string          `json:"action,omitempty"`
}

func (m eventMarket) IsTrade() bool {
	return m.ID != "" && m.Timestamp != nil
}

func (m eventMarket) Trade() eventMarketTrade {
	return eventMarketTrade{
		eventMarketCommon: m.eventMarketCommon,
		ID:                m.ID,
		Timestamp:         *m.Timestamp,
		Quantity:          m.Quantity,
	}
}

func (m eventMarket) IsChange() bool {
	return m.Action != ""
}

func (m eventMarket) Change() eventMarketChange {
	return eventMarketChange{
		eventMarketCommon: m.eventMarketCommon,
		Remaining:         m.Remaining,
		Delta:             m.Delta,
		Action:            m.Action,
	}
}

type eventMarketProduct struct {
	Base  string `json:"base"`
	Quote string `json:"quote"`
}

type eventMarketCommon struct {
	Market eventMarketProduct `json:"market"`
	Side   string             `json:"side"`
	Price  decimal.Decimal    `json:"price"`
}

type eventMarketTrade struct {
	eventMarketCommon
	ID        string          `json:"trade_id"`
	Timestamp xiferTime       `json:"timestamp"`
	Quantity  decimal.Decimal `json:"decimal"`
}

type eventMarketChange struct {
	eventMarketCommon
	Remaining decimal.Decimal `json:"remaining"`
	Delta     decimal.Decimal `json:"delta"`
	Action    string          `json:"action"`
}

type xiferTime time.Time

func (t xiferTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339Nano))
}

func (t *xiferTime) UnmarshalJSON(data []byte) error {
	var formatted string

	if err := json.Unmarshal(data, &formatted); err != nil {
		return err
	}

	tm, err := time.Parse(time.RFC3339Nano, formatted)
	if err != nil {
		return err
	}

	*t = xiferTime(tm)

	return nil
}
