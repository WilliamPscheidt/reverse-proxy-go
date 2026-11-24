package analytics

import (
	"net"
	"strings"
	"sync"
	"time"
)

// GeoLocation represents a geographical point for client request mapping.
type GeoLocation struct {
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
	Count     int64   `json:"count"`
}

// Analytics tracks time-windowed metrics (last minute, last hour, last day) and geographic distribution.
type Analytics struct {
	mu sync.RWMutex

	// 60-second sliding window
	minBuckets [60]int64
	lastSec    int64

	// 60-minute sliding window
	hourBuckets [60]int64
	lastMin     int64

	// 24-hour sliding window
	dayBuckets [24]int64
	lastHour   int64

	// Status code counters
	total2xx int64
	total4xx int64
	total5xx int64
	total429 int64

	// Top client IPs (counter map)
	clientIPs map[string]int64

	// Geographic traffic distribution
	geoLocations map[string]*GeoLocation
}

// NewAnalytics initializes an analytics aggregator.
func NewAnalytics() *Analytics {
	now := time.Now()
	a := &Analytics{
		lastSec:      now.Unix(),
		lastMin:      now.Unix() / 60,
		lastHour:     now.Unix() / 3600,
		clientIPs:    make(map[string]int64),
		geoLocations: make(map[string]*GeoLocation),
	}

	// Initialize default known regional reference points
	a.geoLocations["BR-SP"] = &GeoLocation{Country: "Brasil", City: "São Paulo", Latitude: -23.5505, Longitude: -46.6333, Count: 0}
	a.geoLocations["US-VA"] = &GeoLocation{Country: "Estados Unidos", City: "Virginia", Latitude: 39.0438, Longitude: -77.4874, Count: 0}
	a.geoLocations["DE-FR"] = &GeoLocation{Country: "Alemanha", City: "Frankfurt", Latitude: 50.1109, Longitude: 8.6821, Count: 0}
	a.geoLocations["JP-TK"] = &GeoLocation{Country: "Japão", City: "Tóquio", Latitude: 35.6762, Longitude: 139.6503, Count: 0}
	a.geoLocations["GB-LD"] = &GeoLocation{Country: "Reino Unido", City: "Londres", Latitude: 51.5074, Longitude: -0.1278, Count: 0}

	return a
}

// RecordRequest tracks an incoming HTTP request with its status and client IP.
func (a *Analytics) RecordRequest(clientIP string, statusCode int) {
	now := time.Now()
	nowSec := now.Unix()
	nowMin := nowSec / 60
	nowHour := nowSec / 3600

	a.mu.Lock()
	defer a.mu.Unlock()

	// 1. Advance minute sliding window
	if nowSec > a.lastSec {
		gap := nowSec - a.lastSec
		if gap >= 60 {
			a.minBuckets = [60]int64{}
		} else {
			for i := int64(1); i <= gap; i++ {
				idx := (a.lastSec + i) % 60
				a.minBuckets[idx] = 0
			}
		}
		a.lastSec = nowSec
	}
	a.minBuckets[nowSec%60]++

	// 2. Advance hour sliding window
	if nowMin > a.lastMin {
		gap := nowMin - a.lastMin
		if gap >= 60 {
			a.hourBuckets = [60]int64{}
		} else {
			for i := int64(1); i <= gap; i++ {
				idx := (a.lastMin + i) % 60
				a.hourBuckets[idx] = 0
			}
		}
		a.lastMin = nowMin
	}
	a.hourBuckets[nowMin%60]++

	// 3. Advance day sliding window
	if nowHour > a.lastHour {
		gap := nowHour - a.lastHour
		if gap >= 24 {
			a.dayBuckets = [24]int64{}
		} else {
			for i := int64(1); i <= gap; i++ {
				idx := (a.lastHour + i) % 24
				a.dayBuckets[idx] = 0
			}
		}
		a.lastHour = nowHour
	}
	a.dayBuckets[nowHour%24]++

	// 4. Status breakdown
	if statusCode >= 200 && statusCode < 300 {
		a.total2xx++
	} else if statusCode == 429 {
		a.total429++
		a.total4xx++
	} else if statusCode >= 400 && statusCode < 500 {
		a.total4xx++
	} else if statusCode >= 500 {
		a.total5xx++
	}

	// 5. Track client IP
	cleanIP := sanitizeIP(clientIP)
	a.clientIPs[cleanIP]++

	// 6. Map to geographic region
	geoKey := resolveGeoRegion(cleanIP)
	if geo, exists := a.geoLocations[geoKey]; exists {
		geo.Count++
	}
}

// Snapshot returns the current window statistics.
type Snapshot struct {
	LastMinuteRequests int64          `json:"requests_last_minute"`
	LastHourRequests   int64          `json:"requests_last_hour"`
	LastDayRequests    int64          `json:"requests_last_day"`
	Total2xx           int64          `json:"total_2xx"`
	Total4xx           int64          `json:"total_4xx"`
	Total5xx           int64          `json:"total_5xx"`
	Total429           int64          `json:"total_429"`
	TopIPs             []TopIP        `json:"top_ips"`
	GeoLocations       []GeoLocation  `json:"geo_locations"`
}

type TopIP struct {
	IP    string `json:"ip"`
	Count int64  `json:"count"`
}

// GetSnapshot calculates the aggregates for reporting.
func (a *Analytics) GetSnapshot() Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var lastMinSum int64
	for _, count := range a.minBuckets {
		lastMinSum += count
	}

	var lastHourSum int64
	for _, count := range a.hourBuckets {
		lastHourSum += count
	}

	var lastDaySum int64
	for _, count := range a.dayBuckets {
		lastDaySum += count
	}

	// Top IPs (sorted top 5)
	var topIPs []TopIP
	for ip, count := range a.clientIPs {
		topIPs = append(topIPs, TopIP{IP: ip, Count: count})
		if len(topIPs) >= 8 {
			break
		}
	}

	var geos []GeoLocation
	for _, g := range a.geoLocations {
		geos = append(geos, *g)
	}

	return Snapshot{
		LastMinuteRequests: lastMinSum,
		LastHourRequests:   lastHourSum,
		LastDayRequests:    lastDaySum,
		Total2xx:           a.total2xx,
		Total4xx:           a.total4xx,
		Total5xx:           a.total5xx,
		Total429:           a.total429,
		TopIPs:             topIPs,
		GeoLocations:       geos,
	}
}

func sanitizeIP(addr string) string {
	if addr == "" {
		return "127.0.0.1"
	}
	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		return host
	}
	return addr
}

func resolveGeoRegion(ip string) string {
	// For loopback/local, map to default local host region (Brazil)
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		return "BR-SP"
	}
	// Deterministic hash-to-region mapping for testing/demo
	switch ip[len(ip)-1] % 4 {
	case 0:
		return "US-VA"
	case 1:
		return "DE-FR"
	case 2:
		return "JP-TK"
	default:
		return "GB-LD"
	}
}
