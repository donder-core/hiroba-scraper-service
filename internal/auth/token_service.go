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

	"github.com/donder-core/hiroba-scraper-service/internal/auth/models"
)

const (
	userAgent       = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Mobile Safari/537.36"
	maxRedirects    = 3 // 3 redirects is enough for the login process
	tokenCookieName = "_token_v2"
)

type TokenService struct {
	client       *http.Client
	logger       *log.Logger
	currentToken string
	tokenMu      sync.RWMutex // Protects currentToken from concurrent access
}

var (
	instance *TokenService
	once     sync.Once
)

// GetInstance returns the singleton instance of TokenService
func GetInstance() *TokenService {
	once.Do(func() {
		jar, err := cookiejar.New(nil)
		if err != nil {
			log.Fatal(err)
		}

		logger := log.New(os.Stdout, "[AUTH] ", log.LstdFlags)

		instance = &TokenService{
			client: &http.Client{
				Jar:           jar,
				CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
			},
			logger: logger,
		}
	})
	return instance
}

// Authenticate performs the complete authentication flow and returns the token cookie value
func (th *TokenService) Authenticate(username, password string) (string, error) {
	th.logger.Printf("Authenticating with Bandai Namco ID")
	redirectURL := th.authenticateBandaiNamco(username, password)

	th.logger.Printf("Following OAuth2 redirects")
	th.followOAuthRedirects(redirectURL)

	th.logger.Printf("Submitting login selection")
	th.submitLoginSelection()

	th.logger.Printf("Verifying authentication")
	if err := th.verifyAuthentication(); err != nil {
		return "", err
	}

	token := th.getToken()
	if token == "" {
		return "", fmt.Errorf("token cookie not found")
	}

	th.logger.Printf("Authentication completed successfully")
	th.SetCurrentToken(token)
	return th.GetCurrentToken(), nil
}

func (th *TokenService) GetCurrentToken() string {
	th.tokenMu.RLock()
	defer th.tokenMu.RUnlock()
	return th.currentToken
}

func (th *TokenService) SetCurrentToken(token string) {
	th.tokenMu.Lock()
	defer th.tokenMu.Unlock()
	th.currentToken = token
}

// GetClient returns the authenticated HTTP client
func (th *TokenService) GetClient() *http.Client {
	return th.client
}

func (th *TokenService) authenticateBandaiNamco(username, password string) string {
	th.logger.Printf("Sending authentication request...")
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

	th.logger.Printf("Received authentication response with status: %d", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var loginResp models.LoginResponse
	if err := json.Unmarshal(bodyBytes, &loginResp); err != nil {
		log.Fatalf("Failed to parse login response: %v", err)
	}

	if loginResp.Status != 0 {
		log.Fatalf("Login failed: %s", loginResp.Message)
	}

	if loginResp.Redirect == "" {
		log.Fatal("No redirect URL in response")
	}

	th.logger.Printf("Authentication successful, redirect URL received")
	return loginResp.Redirect
}

func (th *TokenService) followOAuthRedirects(startURL string) {
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
		th.logger.Printf("Completed redirect chain after %d redirects", i+1)
		return
	}

	log.Fatal("Too many redirects")
}

func (th *TokenService) submitLoginSelection() {
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

func (th *TokenService) verifyAuthentication() error {
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
		th.logger.Printf("Authentication verification successful")
		return nil
	}

	th.logger.Printf("Authentication verification failed - login indicators not found")
	return fmt.Errorf("authentication verification failed")
}

func (th *TokenService) getToken() string {
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
