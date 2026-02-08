package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
)

const (
	userAgent       = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Mobile Safari/537.36"
	maxRedirects    = 3 // 3 redirects is enough for the login process
	tokenCookieName = "_token_v2"
)

type TokenHandler struct {
	client *http.Client
	logger *log.Logger
	debug  bool
}

type LoginResponse struct {
	Status   int    `json:"status"`
	Redirect string `json:"redirect"`
	Message  string `json:"message"`
}

var (
	instance *TokenHandler
	once     sync.Once
)

// GetInstance returns the singleton instance of TokenHandler with debug mode disabled
func GetInstance() *TokenHandler {
	return GetInstanceWithDebug(false)
}

// GetInstanceWithDebug returns the singleton instance of TokenHandler with configurable debug mode
func GetInstanceWithDebug(debug bool) *TokenHandler {
	once.Do(func() {
		jar, err := cookiejar.New(nil)
		if err != nil {
			log.Fatal(err)
		}

		logger := log.New(os.Stdout, "[AUTH] ", log.LstdFlags)

		instance = &TokenHandler{
			client: &http.Client{
				Jar:           jar,
				CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
			},
			logger: logger,
			debug:  debug,
		}
	})
	return instance
}

// SetDebug enables or disables debug logging
func (th *TokenHandler) SetDebug(debug bool) {
	th.debug = debug
}

// logDebug logs a message only if debug mode is enabled
func (th *TokenHandler) logDebug(format string, v ...interface{}) {
	if th.debug {
		th.logger.Printf(format, v...)
	}
}

// Authenticate performs the complete authentication flow and returns the token cookie value
func (th *TokenHandler) Authenticate(username, password string) (string, error) {
	th.logDebug("Step 1: Authenticating with Bandai Namco ID")
	redirectURL := th.authenticateBandaiNamco(username, password)

	th.logDebug("Step 2: Following OAuth2 redirects")
	th.followOAuthRedirects(redirectURL)

	th.logDebug("Step 3: Submitting login selection")
	th.submitLoginSelection()

	th.logDebug("Step 4: Verifying authentication")
	if err := th.verifyAuthentication(); err != nil {
		return "", err
	}

	token := th.getToken()
	if token == "" {
		return "", fmt.Errorf("token cookie not found")
	}

	th.logDebug("Authentication completed successfully")
	return token, nil
}

// GetClient returns the authenticated HTTP client
func (th *TokenHandler) GetClient() *http.Client {
	return th.client
}

func (th *TokenHandler) authenticateBandaiNamco(username, password string) string {
	formData := url.Values{}
	formData.Set("client_id", "nbgi_taiko")
	formData.Set("redirect_uri", "https://www.bandainamcoid.com/v2/oauth2/auth?back=v3&client_id=nbgi_taiko&scope=JpGroupAll&redirect_uri=https%3A%2F%2Fdonderhiroba.jp%2Flogin_process.php")
	formData.Set("backto", "")
	formData.Set("customize_id", "")
	formData.Set("login_id", username)
	formData.Set("password", password)
	formData.Set("retention", "1")
	formData.Set("language", "en")
	formData.Set("cookie", `{"language":"en","retention":"0"}`)
	formData.Set("prompt", "login")

	req, err := http.NewRequest("POST", "https://account-api.bandainamcoid.com/v3/login/idpw", strings.NewReader(formData.Encode()))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("origin", "https://account.bandainamcoid.com")
	req.Header.Set("user-agent", userAgent)

	resp, err := th.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(bodyBytes, &loginResp); err != nil {
		log.Fatalf("Failed to parse login response: %v", err)
	}

	if loginResp.Status != 0 {
		log.Fatalf("Login failed: %s", loginResp.Message)
	}

	if loginResp.Redirect == "" {
		log.Fatal("No redirect URL in response")
	}

	return loginResp.Redirect
}

func (th *TokenHandler) followOAuthRedirects(startURL string) {
	currentURL := startURL

	for i := 0; i < maxRedirects; i++ {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("user-agent", userAgent)

		resp, err := th.client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			currentURL = resp.Header.Get("Location")
			resp.Body.Close()
			if currentURL == "" {
				log.Fatal("Redirect without Location header")
			}
			continue
		}

		resp.Body.Close()
		return
	}

	log.Fatal("Too many redirects")
}

func (th *TokenHandler) submitLoginSelection() {
	selectData := strings.NewReader("id_pos=1&mode=exec")
	req, err := http.NewRequest("POST", "https://donderhiroba.jp/login_select.php", selectData)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("user-agent", userAgent)
	req.Header.Set("referer", "https://donderhiroba.jp/login_select.php")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("origin", "https://donderhiroba.jp")

	resp, err := th.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body)
}

func (th *TokenHandler) verifyAuthentication() error {
	req, err := http.NewRequest("GET", "https://donderhiroba.jp/index.php", nil)
	if err != nil {
		return err
	}

	req.Header.Set("user-agent", userAgent)
	req.Header.Set("referer", "https://donderhiroba.jp/login_select.php")

	resp, err := th.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pageContent := string(bodyBytes)
	if strings.Contains(pageContent, "logout") || strings.Contains(pageContent, "マイページ") {
		th.logDebug("Authentication verification successful")
		return nil
	}

	th.logDebug("Authentication verification failed - login indicators not found")
	return fmt.Errorf("authentication verification failed")
}

func (th *TokenHandler) getToken() string {
	dondonURL, err := url.Parse("https://donderhiroba.jp")
	if err != nil {
		return ""
	}

	cookies := th.client.Jar.Cookies(dondonURL)
	for _, cookie := range cookies {
		if cookie.Name == tokenCookieName {
			return cookie.Value
		}
	}

	return ""
}
