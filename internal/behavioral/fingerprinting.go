package behavioral

import (
	"crypto/tls"
	"math/rand"
	"sync"
	"time"
)

// FingerprintingProtection provides protection against various fingerprinting techniques
type FingerprintingProtection struct {
	tlsFingerprinter  *TLSFingerprinter
	httpFingerprinter *HTTPFingerprinter
	browserMasker    *BrowserMasker
	mu               sync.RWMutex
	enabled          bool
	protectionLevel  int // 1-5 scale
}

// TLSFingerprinter handles TLS fingerprint protection
type TLSFingerprinter struct {
	supportedVersions    []uint16
	supportedCiphers     []uint16
	extensions          []uint16
	alpnProtocols       []string
	currentProfile      string
	profileRotationTime time.Duration
	lastRotation        time.Time
	mu                  sync.RWMutex
}

// HTTPFingerprinter handles HTTP fingerprint protection
type HTTPFingerprinter struct {
	userAgents       []string
	acceptLanguages  []string
	acceptEncodings  []string
	connectionTypes  []string
	upgradeInsecure  bool
	currentProfile    string
	profileRotation   time.Duration
	lastRotation      time.Time
	mu                sync.RWMutex
}

// BrowserMasker handles browser behavior masking
type BrowserMasker struct {
	requestPatterns   map[string][]time.Duration
	headerRandomization bool
	cookieManagement   bool
	jsObfuscation      bool
	canvasFingerprinting bool
	currentBrowser     string
	mu                 sync.RWMutex
}

// NewFingerprintingProtection creates new fingerprinting protection
func NewFingerprintingProtection() *FingerprintingProtection {
	return &FingerprintingProtection{
		tlsFingerprinter:  NewTLSFingerprinter(),
		httpFingerprinter: NewHTTPFingerprinter(),
		browserMasker:     NewBrowserMasker(),
		enabled:           true,
		protectionLevel:   3, // Medium protection by default
	}
}

// NewTLSFingerprinter creates new TLS fingerprinter
func NewTLSFingerprinter() *TLSFingerprinter {
	return &TLSFingerprinter{
		supportedVersions: []uint16{
			tls.VersionTLS12,
			tls.VersionTLS13,
		},
		supportedCiphers: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		extensions: []uint16{
			0x002b, // supported_versions
			0x0010, // application_layer_protocol_negotiation
			0x000a, // supported_groups
			0x000d, // signature_algorithms
			0x0033, // key_share
			0x002d, // psk_key_exchange_modes
			0x002a, // early_data
		},
		alpnProtocols:       []string{"h2", "http/1.1"},
		currentProfile:      "chrome",
		profileRotationTime: 30 * time.Minute,
		lastRotation:        time.Now(),
	}
}

// NewHTTPFingerprinter creates new HTTP fingerprinter
func NewHTTPFingerprinter() *HTTPFingerprinter {
	return &HTTPFingerprinter{
		userAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/121.0",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		acceptLanguages: []string{
			"en-US,en;q=0.9",
			"en-GB,en;q=0.9,en;q=0.8",
			"ru-RU,ru;q=0.9,en;q=0.8",
			"zh-CN,zh;q=0.9,en;q=0.8",
		},
		acceptEncodings: []string{
			"gzip, deflate, br",
			"gzip, deflate",
			"br, gzip, deflate",
		},
		connectionTypes: []string{
			"keep-alive",
			"close",
		},
		upgradeInsecure:    true,
		currentProfile:     "chrome",
		profileRotation:    45 * time.Minute,
		lastRotation:       time.Now(),
	}
}

// NewBrowserMasker creates new browser masker
func NewBrowserMasker() *BrowserMasker {
	return &BrowserMasker{
		requestPatterns: map[string][]time.Duration{
			"normal": {
				50 * time.Millisecond,
				100 * time.Millisecond,
				200 * time.Millisecond,
			},
			"mobile": {
				30 * time.Millisecond,
				80 * time.Millisecond,
				150 * time.Millisecond,
			},
			"bot": {
				5 * time.Millisecond,
				10 * time.Millisecond,
				20 * time.Millisecond,
			},
		},
		headerRandomization:  true,
		cookieManagement:     true,
		jsObfuscation:       true,
		canvasFingerprinting: true,
		currentBrowser:       "chrome",
	}
}

// ProtectTLSFingerprint protects TLS fingerprint
func (fp *FingerprintingProtection) ProtectTLSFingerprint(config *tls.Config) error {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if !fp.enabled {
		return nil
	}

	return fp.tlsFingerprinter.ApplyProtection(config)
}

// ApplyProtection applies TLS fingerprint protection
func (tf *TLSFingerprinter) ApplyProtection(config *tls.Config) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	// Check if profile rotation is needed
	if time.Since(tf.lastRotation) > tf.profileRotationTime {
		tf.rotateProfile()
	}

	// Apply fingerprint based on current profile
	switch tf.currentProfile {
	case "chrome":
		return tf.applyChromeProfile(config)
	case "firefox":
		return tf.applyFirefoxProfile(config)
	case "safari":
		return tf.applySafariProfile(config)
	case "edge":
		return tf.applyEdgeProfile(config)
	default:
		return tf.applyGenericProfile(config)
	}
}

// applyChromeProfile applies Chrome TLS profile
func (tf *TLSFingerprinter) applyChromeProfile(config *tls.Config) error {
	config.MinVersion = tls.VersionTLS12
	config.MaxVersion = tls.VersionTLS13
	
	// Chrome-specific cipher suite order
	config.CipherSuites = []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
	
	// Chrome-specific extensions
	config.NextProtos = []string{"h2", "http/1.1"}
	
	return nil
}

// applyFirefoxProfile applies Firefox TLS profile
func (tf *TLSFingerprinter) applyFirefoxProfile(config *tls.Config) error {
	config.MinVersion = tls.VersionTLS12
	config.MaxVersion = tls.VersionTLS13
	
	// Firefox-specific cipher suite order
	config.CipherSuites = []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	}
	
	config.NextProtos = []string{"h2", "http/1.1"}
	
	return nil
}

// applySafariProfile applies Safari TLS profile
func (tf *TLSFingerprinter) applySafariProfile(config *tls.Config) error {
	config.MinVersion = tls.VersionTLS12
	config.MaxVersion = tls.VersionTLS13
	
	// Safari-specific cipher suite order
	config.CipherSuites = []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
	
	config.NextProtos = []string{"h2", "http/1.1"}
	
	return nil
}

// applyEdgeProfile applies Edge TLS profile
func (tf *TLSFingerprinter) applyEdgeProfile(config *tls.Config) error {
	// Edge is similar to Chrome but with slight variations
	return tf.applyChromeProfile(config)
}

// applyGenericProfile applies generic TLS profile
func (tf *TLSFingerprinter) applyGenericProfile(config *tls.Config) error {
	config.MinVersion = tls.VersionTLS12
	config.MaxVersion = tls.VersionTLS13
	config.CipherSuites = tf.supportedCiphers
	config.NextProtos = tf.alpnProtocols
	
	return nil
}

// rotateProfile rotates TLS profile for fingerprint protection
func (tf *TLSFingerprinter) rotateProfile() {
	profiles := []string{"chrome", "firefox", "safari", "edge", "generic"}
	tf.currentProfile = profiles[rand.Intn(len(profiles))]
	tf.lastRotation = time.Now()
}

// ProtectHTTPHeaders protects HTTP headers from fingerprinting
func (fp *FingerprintingProtection) ProtectHTTPHeaders(headers map[string]string) error {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if !fp.enabled {
		return nil
	}

	return fp.httpFingerprinter.ApplyProtection(headers)
}

// ApplyProtection applies HTTP fingerprint protection
func (hf *HTTPFingerprinter) ApplyProtection(headers map[string]string) error {
	hf.mu.Lock()
	defer hf.mu.Unlock()

	// Check if profile rotation is needed
	if time.Since(hf.lastRotation) > hf.profileRotation {
		hf.rotateProfile()
	}

	// Apply fingerprint based on current profile
	switch hf.currentProfile {
	case "chrome":
		return hf.applyChromeProfile(headers)
	case "firefox":
		return hf.applyFirefoxProfile(headers)
	case "safari":
		return hf.applySafariProfile(headers)
	case "edge":
		return hf.applyEdgeProfile(headers)
	default:
		return hf.applyGenericProfile(headers)
	}
}

// applyChromeProfile applies Chrome HTTP profile
func (hf *HTTPFingerprinter) applyChromeProfile(headers map[string]string) error {
	// Set Chrome User-Agent
	headers["User-Agent"] = hf.userAgents[0]
	
	// Set Chrome-specific headers
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
	headers["Accept-Language"] = hf.acceptLanguages[0]
	headers["Accept-Encoding"] = hf.acceptEncodings[0]
	headers["Connection"] = hf.connectionTypes[0]
	headers["Upgrade-Insecure-Requests"] = "1"
	headers["Sec-Fetch-Dest"] = "document"
	headers["Sec-Fetch-Mode"] = "navigate"
	headers["Sec-Fetch-Site"] = "none"
	headers["Sec-Fetch-User"] = "?1"
	headers["Sec-Ch-Ua"] = `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`
	headers["Sec-Ch-Ua-Mobile"] = "?0"
	headers["Sec-Ch-Ua-Platform"] = `"Windows"`
	
	return nil
}

// applyFirefoxProfile applies Firefox HTTP profile
func (hf *HTTPFingerprinter) applyFirefoxProfile(headers map[string]string) error {
	// Set Firefox User-Agent
	headers["User-Agent"] = hf.userAgents[2]
	
	// Set Firefox-specific headers
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
	headers["Accept-Language"] = hf.acceptLanguages[0]
	headers["Accept-Encoding"] = hf.acceptEncodings[0]
	headers["Connection"] = hf.connectionTypes[0]
	headers["Upgrade-Insecure-Requests"] = "1"
	headers["Sec-Fetch-Dest"] = "document"
	headers["Sec-Fetch-Mode"] = "navigate"
	headers["Sec-Fetch-Site"] = "none"
	headers["Sec-Fetch-User"] = "?1"
	
	return nil
}

// applySafariProfile applies Safari HTTP profile
func (hf *HTTPFingerprinter) applySafariProfile(headers map[string]string) error {
	// Set Safari User-Agent
	headers["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15"
	
	// Set Safari-specific headers
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	headers["Accept-Language"] = "en-US,en;q=0.9"
	headers["Accept-Encoding"] = "gzip, deflate, br"
	headers["Connection"] = "keep-alive"
	headers["Upgrade-Insecure-Requests"] = "1"
	
	return nil
}

// applyEdgeProfile applies Edge HTTP profile
func (hf *HTTPFingerprinter) applyEdgeProfile(headers map[string]string) error {
	// Edge is similar to Chrome but with slight variations
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"
	
	// Set Edge-specific headers
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
	headers["Accept-Language"] = hf.acceptLanguages[0]
	headers["Accept-Encoding"] = hf.acceptEncodings[0]
	headers["Connection"] = hf.connectionTypes[0]
	headers["Upgrade-Insecure-Requests"] = "1"
	headers["Sec-Fetch-Dest"] = "document"
	headers["Sec-Fetch-Mode"] = "navigate"
	headers["Sec-Fetch-Site"] = "none"
	headers["Sec-Fetch-User"] = "?1"
	headers["Sec-Ch-Ua"] = `"Not_A Brand";v="8", "Chromium";v="120", "Microsoft Edge";v="120"`
	headers["Sec-Ch-Ua-Mobile"] = "?0"
	headers["Sec-Ch-Ua-Platform"] = `"Windows"`
	
	return nil
}

// applyGenericProfile applies generic HTTP profile
func (hf *HTTPFingerprinter) applyGenericProfile(headers map[string]string) error {
	headers["User-Agent"] = hf.userAgents[rand.Intn(len(hf.userAgents))]
	headers["Accept-Language"] = hf.acceptLanguages[rand.Intn(len(hf.acceptLanguages))]
	headers["Accept-Encoding"] = hf.acceptEncodings[rand.Intn(len(hf.acceptEncodings))]
	headers["Connection"] = hf.connectionTypes[rand.Intn(len(hf.connectionTypes))]
	
	return nil
}

// rotateProfile rotates HTTP profile for fingerprint protection
func (hf *HTTPFingerprinter) rotateProfile() {
	profiles := []string{"chrome", "firefox", "safari", "edge", "generic"}
	hf.currentProfile = profiles[rand.Intn(len(profiles))]
	hf.lastRotation = time.Now()
}

// MaskBrowserBehavior masks browser behavior patterns
func (fp *FingerprintingProtection) MaskBrowserBehavior(requestType string) time.Duration {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if !fp.enabled {
		return 0
	}

	return fp.browserMasker.ApplyMasking(requestType)
}

// ApplyMasking applies browser behavior masking
func (bm *BrowserMasker) ApplyMasking(requestType string) time.Duration {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	patterns, exists := bm.requestPatterns[bm.currentBrowser]
	if !exists {
		patterns = bm.requestPatterns["normal"]
	}

	if len(patterns) == 0 {
		return 0
	}

	// Add random variation to make timing less predictable
	baseDelay := patterns[rand.Intn(len(patterns))]
	variation := time.Duration(rand.Intn(20)) * time.Millisecond
	
	return baseDelay + variation
}

// UpdateEffectiveness updates fingerprinting protection effectiveness
func (fp *FingerprintingProtection) UpdateEffectiveness(success bool) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if success {
		// Increase protection level if successful
		if fp.protectionLevel < 5 {
			fp.protectionLevel++
		}
	} else {
		// Decrease protection level if detected
		if fp.protectionLevel > 1 {
			fp.protectionLevel--
		}
	}
}

// SetProtectionLevel sets protection level
func (fp *FingerprintingProtection) SetProtectionLevel(level int) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	
	if level >= 1 && level <= 5 {
		fp.protectionLevel = level
	}
}

// GetProtectionLevel returns current protection level
func (fp *FingerprintingProtection) GetProtectionLevel() int {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	
	return fp.protectionLevel
}

// Enable enables fingerprinting protection
func (fp *FingerprintingProtection) Enable() {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.enabled = true
}

// Disable disables fingerprinting protection
func (fp *FingerprintingProtection) Disable() {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.enabled = false
}

// SetEnabled enables or disables fingerprinting protection
func (fp *FingerprintingProtection) SetEnabled(enabled bool) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.enabled = enabled
}

// IsEnabled returns whether fingerprinting protection is enabled
func (fp *FingerprintingProtection) IsEnabled() bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.enabled
}
