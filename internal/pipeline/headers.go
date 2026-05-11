package pipeline

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type HeadersModifier struct {
	RandomHeaders    bool
	CustomHeaders    map[string]string
	UserAgents       []string
	ModifyHost       bool
	AddJunkHeaders   bool
	rand             *rand.Rand
}

func (h *HeadersModifier) Name() string {
	return "headers"
}

func (h *HeadersModifier) Configure(config map[string]interface{}) error {
	if randomHeaders, ok := config["random_headers"].(bool); ok {
		h.RandomHeaders = randomHeaders
	}

	if customHeaders, ok := config["custom_headers"].(map[string]interface{}); ok {
		h.CustomHeaders = make(map[string]string)
		for k, v := range customHeaders {
			if str, ok := v.(string); ok {
				h.CustomHeaders[k] = str
			}
		}
	}

	if userAgents, ok := config["user_agents"].([]interface{}); ok {
		h.UserAgents = make([]string, len(userAgents))
		for i, ua := range userAgents {
			if str, ok := ua.(string); ok {
				h.UserAgents[i] = str
			}
		}
	} else {
		// Default user agents
		h.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		}
	}

	if modifyHost, ok := config["modify_host"].(bool); ok {
		h.ModifyHost = modifyHost
	}

	if addJunk, ok := config["add_junk_headers"].(bool); ok {
		h.AddJunkHeaders = addJunk
	}

	h.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return nil
}

func (h *HeadersModifier) Process(data []byte, direction Direction) []byte {
	// Only process HTTP requests (outbound traffic)
	if direction != DirectionOutbound {
		return data
	}

	// Check if this looks like HTTP
	if !h.isHTTP(data) {
		return data
	}

	// Parse HTTP request
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		// If parsing fails, try simple string manipulation
		return h.modifyHTTPHeaders(data)
	}

	// Modify headers
	h.modifyRequest(req)

	// Rebuild request
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %s HTTP/1.1\r\n", req.Method, req.URL.Path)
	req.Header.Write(&buf)
	buf.WriteString("\r\n")
	
	// Add body if exists
	if req.Body != nil {
		buf.ReadFrom(req.Body)
	}

	return buf.Bytes()
}

func (h *HeadersModifier) isHTTP(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	
	methods := []string{"GET ", "POST", "PUT ", "HEAD", "DELE", "OPTI", "PATC", "CONN"}
	dataStr := string(data[:4])
	
	for _, method := range methods {
		if strings.HasPrefix(dataStr, method) {
			return true
		}
	}
	
	return false
}

func (h *HeadersModifier) modifyRequest(req *http.Request) {
	// Random User-Agent
	if len(h.UserAgents) > 0 {
		ua := h.UserAgents[h.rand.Intn(len(h.UserAgents))]
		req.Header.Set("User-Agent", ua)
	}

	// Add custom headers
	for k, v := range h.CustomHeaders {
		req.Header.Set(k, v)
	}

	// Add random headers
	if h.RandomHeaders {
		h.addRandomHeaders(req)
	}

	// Add junk headers
	if h.AddJunkHeaders {
		h.addJunkHeaders(req)
	}

	// Modify Host header
	if h.ModifyHost {
		h.modifyHostHeader(req)
	}
}

func (h *HeadersModifier) addRandomHeaders(req *http.Request) {
	randomHeaders := map[string][]string{
		"Accept": {
			"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"application/json, text/plain, */*",
			"text/html,application/xml,application/xhtml+xml,text/html;q=0.9,text/plain;q=0.8,image/png,*/*;q=0.5",
		},
		"Accept-Language": {
			"en-US,en;q=0.9",
			"ru-RU,ru;q=0.9,en;q=0.8",
			"en-GB,en-US;q=0.9,en;q=0.8",
		},
		"Accept-Encoding": {
			"gzip, deflate, br",
			"gzip, deflate",
			"br, gzip, deflate",
		},
	}

	for header, values := range randomHeaders {
		if req.Header.Get(header) == "" {
			value := values[h.rand.Intn(len(values))]
			req.Header.Set(header, value)
		}
	}
}

func (h *HeadersModifier) addJunkHeaders(req *http.Request) {
	type junkHeader struct {
		key    string
		values []string
	}
	
	junkHeaders := []junkHeader{
		{"X-Requested-With", []string{"XMLHttpRequest", "Fetch"}},
		{"X-Forwarded-For", []string{fmt.Sprintf("192.168.%d.%d", h.rand.Intn(255)+1, h.rand.Intn(255)+1)}},
		{"X-Real-IP", []string{fmt.Sprintf("10.0.%d.%d", h.rand.Intn(255)+1, h.rand.Intn(255)+1)}},
		{"X-Client-IP", []string{fmt.Sprintf("172.16.%d.%d", h.rand.Intn(255)+1, h.rand.Intn(255)+1)}},
		{"DNT", []string{"1", "0"}},
		{"Sec-Fetch-Dest", []string{"document", "empty", "script"}},
		{"Sec-Fetch-Mode", []string{"navigate", "cors", "no-cors"}},
		{"Sec-Fetch-Site", []string{"none", "same-origin", "cross-site"}},
	}

	for _, junk := range junkHeaders {
		if req.Header.Get(junk.key) == "" && h.rand.Float32() < 0.3 { // 30% chance
			value := junk.values[h.rand.Intn(len(junk.values))]
			req.Header.Set(junk.key, value)
		}
	}
}

func (h *HeadersModifier) modifyHostHeader(req *http.Request) {
	if host := req.Header.Get("Host"); host != "" {
		// Add random case variations
		var newHost []byte
		for _, c := range host {
			char := byte(c)
			if h.rand.Float32() < 0.1 { // 10% chance to change case
				if char >= 'a' && char <= 'z' {
					char = char - 32 // uppercase
				} else if char >= 'A' && char <= 'Z' {
					char = char + 32 // lowercase
				}
			}
			newHost = append(newHost, char)
		}
		req.Header.Set("Host", string(newHost))
	}
}

func (h *HeadersModifier) modifyHTTPHeaders(data []byte) []byte {
	// Simple string-based modification for when HTTP parsing fails
	dataStr := string(data)
	
	// Add random headers if not present
	if !strings.Contains(dataStr, "User-Agent:") && len(h.UserAgents) > 0 {
		ua := h.UserAgents[h.rand.Intn(len(h.UserAgents))]
		dataStr = strings.Replace(dataStr, "\r\n", fmt.Sprintf("\r\nUser-Agent: %s\r\n", ua), 1)
	}
	
	return []byte(dataStr)
}
