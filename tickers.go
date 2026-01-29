package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// WIKI_URL is the source of truth for the S&P 500 components
const WIKI_URL = "https://en.wikipedia.org/wiki/List_of_S%26P_500_companies"

// GetMarketTickers is the main entry point.
// It returns two lists:
// 1. vips: Top ~50 stocks (High Market Cap) -> These get Real-Time WebSockets.
// 2. slow: The remaining ~450 stocks -> These get Background Polling.
func GetMarketTickers() ([]string, []string) {
	fmt.Println("🌍 Attempting to fetch live S&P 500 from Wikipedia...")

	// 1. Try to Scrape
	scrapedList, err := scrapeWikipedia()
	if err != nil {
		fmt.Printf("⚠️ Scrape failed (%v). Switching to Backup Logic.\n", err)
		scrapedList = BACKUP_SP500
	} else {
		fmt.Printf("✅ Success! Found %d companies live.\n", len(scrapedList))
	}

	// 2. The "VIP Filter"
	// Wikipedia is sorted Alphabetically, but we want the "Live" feed to be
	// the most popular stocks (Market Cap), not just stocks starting with 'A'.
	// We use a curated list of the Top 50 to force them into the VIP lane.

	// A map for fast lookup of what we've already assigned to VIP
	vipSet := make(map[string]bool)
	var vips []string

	// Force our Heavy Hitters into the VIP list first
	for _, sym := range CURATED_VIPS {
		vipSet[sym] = true
		vips = append(vips, sym)
	}

	// 3. Fill the "Slow" list with everyone else
	var slow []string
	for _, sym := range scrapedList {
		// If this stock is NOT in our VIP list, put it in the polling list
		if !vipSet[sym] {
			slow = append(slow, sym)
		}
	}

	return vips, slow
}

// scrapeWikipedia parses the official table
func scrapeWikipedia() ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Create a Custom Request
	req, err := http.NewRequest("GET", WIKI_URL, nil)
	if err != nil {
		return nil, err
	}

	// 2. Set the User-Agent to mimic a Chrome Browser
	// This prevents the 403 Forbidden error
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 3. Execute the Request
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("wikipedia returned status: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var tickers []string
	// The first table with id "constituents" contains the list
	doc.Find("#constituents tbody tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip header
		}

		symbol := s.Find("td:nth-child(1)").Text()
		symbol = strings.TrimSpace(symbol)

		// CRITICAL FIX: Convert "BRK.B" -> "BRK-B"
		// If you don't do this, Yahoo Finance (the polling engine) will crash on Berkshire Hathaway.
		symbol = strings.ReplaceAll(symbol, ".", "-")

		if symbol != "" {
			tickers = append(tickers, symbol)
		}
	})

	if len(tickers) < 450 {
		return nil, fmt.Errorf("scraped list too short (%d), suspecting html change", len(tickers))
	}

	return tickers, nil
}

// CURATED_VIPS: The Top 50 by Weight (Approx 2025/2026)
// These are ALWAYS streamed via WebSocket for maximum visual impact.
var CURATED_VIPS = []string{
	"AAPL", "MSFT", "NVDA", "AMZN", "GOOGL", "META", "TSLA", "BRK.B", "LLY", "AVGO",
	"JPM", "UNH", "V", "XOM", "MA", "JNJ", "PG", "HD", "COST", "MRK",
	"ABBV", "CRM", "CVX", "WMT", "AMD", "PEP", "KO", "NFLX", "BAC", "ACN",
	"LIN", "MCD", "DIS", "ADBE", "TMO", "CSCO", "ABT", "TMUS", "QCOM", "INTC",
	"WFC", "CMCSA", "PFE", "VZ", "DHR", "INTU", "IBM", "AMGN", "NKE", "TXN",
}

// BACKUP_SP500: The Safety Net (Generated via Python)
var BACKUP_SP500 = []string{
	"MMM", "AOS", "ABT", "ABBV", "ACN", "ADBE", "AMD", "AES", "AFL", "A",
	"APD", "ABNB", "AKAM", "ALB", "ARE", "ALGN", "ALLE", "LNT", "ALL", "GOOGL",
	"GOOG", "MO", "AMZN", "AMCR", "AEE", "AEP", "AXP", "AIG", "AMT", "AWK",
	"AMP", "AME", "AMGN", "APH", "ADI", "AON", "APA", "APO", "AAPL", "AMAT",
	"APTV", "ACGL", "ADM", "ANET", "AJG", "AIZ", "T", "ATO", "ADSK", "ADP",
	"AZO", "AVB", "AVY", "AXON", "BKR", "BALL", "BAC", "BK", "BBWI", "BAX",
	"BDX", "BRK-B", "BBY", "BIO", "TECH", "BIIB", "BLK", "BX", "BA", "BKNG",
	"BWA", "BXP", "BSX", "BMY", "AVGO", "BR", "BRO", "BF-B", "BLDR", "BG",
	"CDNS", "CZR", "CPT", "COF", "CAH", "KMX", "CCL", "CARR", "CAT", "CBOE",
	"CBRE", "CDW", "CE", "COR", "CNC", "CNP", "CF", "CHRW", "CRL", "SCHW",
	"CHTR", "CVX", "CMG", "CB", "CHD", "CI", "CINF", "CTAS", "CSCO", "C",
	"CFG", "CLX", "CME", "CMS", "KO", "CTSH", "CL", "CMCSA", "CAG", "COP",
	"ED", "STZ", "CEG", "COO", "CPRT", "GLW", "CPB", "CTVA", "CSGP", "COST",
	"CTRA", "CRWD", "CCI", "CSX", "CMI", "CVS", "DHI", "DHR", "DRI", "DVA",
	"DE", "DAL", "DELL", "XRAY", "DVN", "DXCM", "FANG", "DLR", "DFS", "DIS",
	"DG", "DLTR", "D", "DPZ", "DOV", "DOW", "DTE", "DUK", "DD", "EMN",
	"ETN", "EBAY", "ECL", "EIX", "EW", "EA", "ELV", "EMR", "ENPH", "ETR",
	"EOG", "EPAM", "EQT", "EFX", "EQIX", "EQR", "ESS", "EL", "ETSY", "EG",
	"EVRG", "ES", "EXC", "EXPE", "EXPD", "EXR", "XOM", "FFIV", "FDS", "FICO",
	"FAST", "FRT", "FDX", "FIS", "FITB", "FSLR", "FE", "FI", "F", "FTNT",
	"FTV", "FOXA", "FOX", "BEN", "FCX", "GRMN", "IT", "GE", "GEHC", "GEV",
	"GEN", "GNRC", "GD", "GIS", "GM", "GPC", "GILD", "GPN", "GL", "GS",
	"HAL", "HIG", "HAS", "HCA", "DOC", "HSIC", "HSY", "HES", "HPE", "HLT",
	"HOLX", "HD", "HON", "HRL", "HST", "HWM", "HPQ", "HUBB", "HUM", "HBAN",
	"HII", "IBM", "IEX", "IDXX", "ITW", "INCY", "IR", "PODD", "INTC", "ICE",
	"IP", "IPG", "IFF", "INTU", "ISRG", "IVZ", "INVH", "IQV", "IRM", "JBHT",
	"JBL", "JKHY", "J", "JNJ", "JCI", "JPM", "K", "KVH", "KDP", "KEY",
	"KEYS", "KMB", "KIM", "KMI", "KKR", "KLAC", "KHC", "KR", "LHX", "LH",
	"LRCX", "LW", "LVS", "LDOS", "LEN", "LIN", "LLY", "LKQ", "LMT", "L",
	"LOW", "LULU", "LYB", "MTB", "MRO", "MPC", "MKTX", "MAR", "MMC", "MLM",
	"MAS", "MA", "MTCH", "MKC", "MCD", "MCK", "MDT", "MRK", "META", "MET",
	"MTD", "MGM", "MCHP", "MU", "MSFT", "MAA", "MRNA", "MHK", "MOH", "TAP",
	"MDLZ", "MPWR", "MNST", "MCO", "MS", "MOS", "MSI", "MSCI", "NDAQ", "NTAP",
	"NFLX", "NEM", "NWSA", "NWS", "NEE", "NKE", "NI", "NDSN", "NSC", "NTRS",
	"NOC", "NOW", "NRG", "NUE", "NVDA", "NVR", "NXPI", "ORLY", "OXY", "ODFL",
	"OMC", "ON", "OKE", "ORCL", "OTIS", "PCAR", "PKG", "PLTR", "PANW", "PARA",
	"PH", "PAYX", "PAYC", "PYPL", "PNR", "PEP", "PFE", "PCG", "PM", "PSX",
	"PNW", "PXD", "PNC", "POOL", "PPG", "PPL", "PFG", "PG", "PGR", "PLD",
	"PRU", "PEG", "PTC", "PSA", "PHM", "QRVO", "PWR", "QCOM", "DGX", "RL",
	"RJF", "RTX", "O", "REG", "REGN", "RF", "RSG", "RMD", "RVTY", "RHI",
	"ROK", "ROL", "ROP", "ROST", "RCL", "SPGI", "CRM", "SBAC", "SLB", "STX",
	"SRE", "SHW", "SPG", "SWKS", "SJM", "SNA", "SEDG", "SO", "LUV", "SWK",
	"SBUX", "STT", "STLD", "STE", "SYK", "SMCI", "SYF", "SNPS", "SYY", "TMUS",
	"TROW", "TTWO", "TPR", "TRGP", "TGT", "TEL", "TDY", "TFX", "TER", "TSLA",
	"TXN", "TXT", "TMO", "TJX", "TSCO", "TT", "TDG", "TRV", "TRMB", "TFC",
	"TYL", "TSN", "USB", "UBER", "UDR", "ULTA", "UNP", "UAL", "UPS", "URI",
	"UNH", "UHS", "VLO", "VTR", "VLTO", "VRSN", "VRSK", "VZ", "VRTX", "VFC",
	"VTRS", "VICI", "V", "VMC", "WAB", "WBA", "WMT", "DIS", "WBD", "WM",
	"WAT", "WEC", "WFC", "WELL", "WST", "WDC", "WRK", "WY", "WHR", "WMB",
	"WTW", "GWW", "WYNN", "XEL", "XYL", "YUM", "ZBRA", "ZBH", "ZTS",
}
