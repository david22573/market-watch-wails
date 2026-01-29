package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// --- CONFIG ---
// NOTE: Use your own API keys. Finnhub is free for sandbox/limited use.
const FINNHUB_KEY = "d5t7cnpr01qt62nhnj6gd5t7cnpr01qt62nhnj70" // Sandbox/Free key

// --- DATA STRUCTURES ---

type App struct {
	ctx         context.Context
	yahooClient *http.Client
	yahooCrumb  string

	// App State
	useDemoMode bool

	// Dynamic Lists
	// We need a separate mutex for the list because the poller reads it
	// while the search function might append to it.
	listMutex sync.Mutex
	vipList   []string
	pollList  []string
}

type StockState struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
	IsLive bool    `json:"is_live"`
	IsVIP  bool    `json:"is_vip"` // New field for UI sorting
}

// Yahoo API Response Structure
type YahooResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol                     string  `json:"symbol"`
			RegularMarketPrice         float64 `json:"regularMarketPrice"`
			RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteResponse"`
}

// Global State
var market = make(map[string]*StockState)
var marketMutex sync.RWMutex // Protects the 'market' map

// --- APP LIFECYCLE ---

func NewApp() *App {
	jar, _ := cookiejar.New(nil)
	return &App{
		yahooClient: &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		},
		useDemoMode: false,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("[DEBUG] App startup called.")

	// 1. Yahoo Auth
	if err := a.initYahooSession(); err != nil {
		fmt.Printf("[WARNING] Yahoo Auth Failed (%v). Defaulting to DEMO MODE.\n", err)
		a.useDemoMode = true
	} else {
		fmt.Println("[SUCCESS] Authenticated with Yahoo Finance.")
	}

	// 2. Load Initial Data
	vips, slows := getInitialTickers()
	a.vipList = vips
	a.pollList = slows

	// 3. Initialize Map State
	marketMutex.Lock()
	for _, sym := range a.vipList {
		market[sym] = &StockState{Symbol: sym, Price: 0, Change: 0, IsVIP: true}
	}
	for _, sym := range a.pollList {
		market[sym] = &StockState{Symbol: sym, Price: 0, Change: 0, IsVIP: false}
	}
	marketMutex.Unlock()

	// 4. Start Engine
	go a.startMarketEngine()
}

// --- FRONTEND EXPORTED METHODS ---

// TrackTicker adds a new stock to the polling list via the Search bar
func (a *App) TrackTicker(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return "Invalid Symbol"
	}

	marketMutex.Lock()
	if _, exists := market[symbol]; exists {
		marketMutex.Unlock()
		return fmt.Sprintf("%s is already tracked", symbol)
	}
	// Initialize
	market[symbol] = &StockState{Symbol: symbol, Price: 0, Change: 0, IsVIP: false}
	marketMutex.Unlock()

	// Add to Polling List safely
	a.listMutex.Lock()
	a.pollList = append(a.pollList, symbol)
	a.listMutex.Unlock()

	// Trigger immediate fetch for this specific symbol
	go a.fetchBatchYahoo([]string{symbol})

	fmt.Printf("[INFO] Added ticker: %s\n", symbol)
	return fmt.Sprintf("Added %s", symbol)
}

// ToggleDemoMode allows the settings menu to switch data sources
func (a *App) ToggleDemoMode(active bool) {
	a.useDemoMode = active
	state := "LIVE"
	if active {
		state = "DEMO"
	}
	fmt.Printf("[SETTINGS] Switched to %s mode.\n", state)
}

// --- YAHOO AUTH ---

func (a *App) initYahooSession() error {
	// Step 1: Cookie
	req, _ := http.NewRequest("GET", "https://fc.yahoo.com", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := a.yahooClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Step 2: Crumb
	req, _ = http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err = a.yahooClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	a.yahooCrumb = string(body)
	if a.yahooCrumb == "" {
		return fmt.Errorf("empty crumb")
	}
	return nil
}

// --- ENGINE CORE ---

func (a *App) startMarketEngine() {
	// 1. Start WebSocket for VIPs
	go a.runWebSocket(a.vipList)

	// 2. Start Polling for the rest (and user added ones)
	go a.runPolling()

	// 3. Start Broadcaster (Sends data to UI every second)
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			marketMutex.RLock()
			// Convert map to slice for the UI
			list := make([]StockState, 0, len(market))
			for _, v := range market {
				list = append(list, *v)
			}
			marketMutex.RUnlock()

			// Send to Svelte
			runtime.EventsEmit(a.ctx, "market_update", list)
		}
	}
}

// --- ENGINE 1: WEBSOCKET (REAL-TIME VIPs) ---

func (a *App) runWebSocket(targets []string) {
	url := fmt.Sprintf("wss://ws.finnhub.io?token=%s", FINNHUB_KEY)

	for {
		// Reconnect loop
		if a.useDemoMode {
			time.Sleep(1 * time.Second)
			continue // Don't connect if in demo mode
		}

		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// Subscribe
		for _, sym := range targets {
			msg := fmt.Sprintf(`{"type":"subscribe","symbol":"%s"}`, sym)
			c.WriteMessage(websocket.TextMessage, []byte(msg))
		}

		// Read loop
		for {
			if a.useDemoMode {
				c.Close() // Break connection if user toggles demo
				break
			}

			_, msg, err := c.ReadMessage()
			if err != nil {
				break
			}

			var payload struct {
				Type string `json:"type"`
				Data []struct {
					S string  `json:"s"`
					P float64 `json:"p"`
				} `json:"data"`
			}
			json.Unmarshal(msg, &payload)

			if payload.Type == "trade" {
				marketMutex.Lock()
				for _, d := range payload.Data {
					if item, ok := market[d.S]; ok {
						item.Price = d.P
						item.IsLive = true
					}
				}
				marketMutex.Unlock()
			}
		}
		c.Close()
	}
}

// --- ENGINE 2: POLLING (BACKGROUND) ---

func (a *App) runPolling() {
	// Initial Fetch
	a.processPollCycle()

	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.processPollCycle()
		}
	}
}

func (a *App) processPollCycle() {
	// If in Demo Mode, generate fake moves
	if a.useDemoMode {
		a.runDemoGenerator()
		return
	}

	// 1. Get current list safely
	a.listMutex.Lock()
	// Copy the list to avoid holding the lock during HTTP requests
	currentPollList := make([]string, len(a.pollList))
	copy(currentPollList, a.pollList)
	a.listMutex.Unlock()

	// 2. Also refresh VIPs via HTTP just to get the "Change %"
	// (Websockets usually only give Price, not 24h Change)
	allTargets := append(currentPollList, a.vipList...)

	// 3. Batch Fetch
	a.fetchQuotes(allTargets)
}

func (a *App) fetchQuotes(targets []string) {
	chunkSize := 50 // Yahoo URL limit safe-guard
	for i := 0; i < len(targets); i += chunkSize {
		end := i + chunkSize
		if end > len(targets) {
			end = len(targets)
		}

		batch := targets[i:end]
		err := a.fetchBatchYahoo(batch)

		if err != nil {
			fmt.Printf("[ERROR] Batch API Failed: %v\n", err)
			// Optional: Auto-switch to demo if API fails?
			// a.useDemoMode = true
		}

		// Be nice to the API
		time.Sleep(250 * time.Millisecond)
	}
}

func (a *App) fetchBatchYahoo(symbols []string) error {
	if len(symbols) == 0 {
		return nil
	}

	query := strings.Join(symbols, ",")
	endpoint := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s&crumb=%s", query, a.yahooCrumb)

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := a.yahooClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	var data YahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	marketMutex.Lock()
	for _, res := range data.QuoteResponse.Result {
		if item, ok := market[res.Symbol]; ok {
			if res.RegularMarketPrice > 0 {
				item.Price = res.RegularMarketPrice
				item.Change = res.RegularMarketChangePercent
			}
		}
	}
	marketMutex.Unlock()
	return nil
}

// --- ENGINE 3: DEMO GENERATOR ---

func (a *App) runDemoGenerator() {
	marketMutex.Lock()
	defer marketMutex.Unlock()

	for _, item := range market {
		// Initialize price if 0
		if item.Price == 0 {
			item.Price = 50 + rand.Float64()*150
		}

		// Random Walk
		if rand.Float64() < 0.4 { // 40% chance to move per tick
			move := (rand.Float64() - 0.5) * 1.5
			item.Price += move
			item.Change += (move * 0.1)
		}
	}
}

// --- HELPERS ---

func getInitialTickers() ([]string, []string) {
	// VIPs (High refresh rate via WebSocket)
	vips := []string{"NVDA", "TSLA", "AAPL", "AMD", "MSFT", "AMZN", "GOOGL", "META"}

	// Slows (Polled background)
	slows := []string{
		"INTC", "NFLX", "ADBE", "CRM", "CSCO", "PEP", "KO", "JPM", "BAC", "WFC",
		"GS", "MS", "V", "MA", "AXP", "DIS", "CMCSA", "T", "VZ", "TMUS",
		"PFE", "JNJ", "MRK", "ABBV", "LLY", "UNH", "CVS", "WMT", "TGT", "HD",
		"LOW", "NKE", "SBUX", "MCD", "BA", "CAT", "DE", "GE", "MMM", "HON",
		"IBM", "ORCL", "QCOM", "TXN", "AVGO", "MU", "LRCX", "AMAT", "ADI",
		"GILD", "BIIB", "AMGN", "REGN", "VRTX", "ISRG", "SYK", "ZTS", "BDX",
	}
	return vips, slows
}
