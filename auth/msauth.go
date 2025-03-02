package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xmdhs/msauth/auth"
)

const (
	oauth20Token            = `https://login.live.com/oauth20_token.srf`
	authenticateURL         = `https://user.auth.xboxlive.com/user/authenticate`
	authenticatewithXSTSURL = `https://xsts.auth.xboxlive.com/xsts/authorize`
	loginWithXboxURL        = `https://api.minecraftservices.com/authentication/login_with_xbox`
	getTheprofileURL        = `https://api.minecraftservices.com/minecraft/profile`
)

func getCode() (string, error) {
	code, err := auth.Getcode()
	if err != nil {
		return "", fmt.Errorf("getCode: %w", err)
	}
	return code, nil
}

func MsLogin() (*Profile, error) {
	code, err := getCode()
	if err != nil {
		return nil, fmt.Errorf("MsLogin: %w", err)
	}
	token, err := getToken(code)
	if err != nil {
		return nil, fmt.Errorf("MsLogin: %w", err)
	}
	xbltoken, uhs, err := getXbltoken(token)
	if err != nil {
		return nil, fmt.Errorf("MsLogin: %w", err)
	}
	xststoken, err := getXSTStoken(xbltoken)
	if err != nil {
		return nil, fmt.Errorf("MsLogin: %w", err)
	}
	AccessToken, err := loginWithXbox(uhs, xststoken)
	p, err := GetProfile(AccessToken)
	if err != nil {
		return nil, fmt.Errorf("MsLogin: %w", err)
	}
	p.AccessToken = AccessToken
	return p, nil
}

func getToken(code string) (string, error) {
	code = url.QueryEscape(code)
	msg := `client_id=00000000402b5328&code=` + code + `&grant_type=authorization_code&redirect_uri=https://login.live.com/oauth20_desktop.srf&scope=service::user.auth.xboxlive.com::MBI_SSL`
	b, err := httPost(oauth20Token, msg, `application/x-www-form-urlencoded`)
	if err != nil {
		return "", fmt.Errorf("getToken: %w", err)
	}
	var t token
	err = json.Unmarshal(b, &t)
	if err != nil {
		return "", fmt.Errorf("getToken: %w", err)
	}
	if t.AccessToken == "" {
		return "", ErrCode
	}
	return t.AccessToken, nil
}

func getXbltoken(token string) (Xbltoken, uhs string, err error) {
	msg := `{"Properties": {"AuthMethod": "RPS","SiteName": "user.auth.xboxlive.com","RpsTicket": "` + jsonEscape(token) + `"},"RelyingParty": "http://auth.xboxlive.com","TokenType": "JWT"}`
	b, err := httPost(authenticateURL, msg, `application/json`)
	if err != nil {
		return "", "", fmt.Errorf("getXbltoken: %w", err)
	}
	m := msauth{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return "", "", fmt.Errorf("getXbltoken: %w", err)
	}
	if len(m.DisplayClaims.Xui) < 1 {
		return "", "", ErrToken
	}
	return m.Token, m.DisplayClaims.Xui[0].Uhs, nil
}

func getXSTStoken(Xbltoken string) (string, error) {
	msg := `{
		"Properties": {
			"SandboxId": "RETAIL",
			"UserTokens": [
				"` + jsonEscape(Xbltoken) + `" 
			]
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType": "JWT"
	 }`
	b, err := httPost(authenticatewithXSTSURL, msg, `application/json`)
	if err != nil {
		return "", fmt.Errorf("getXSTStoken: %w", err)
	}
	m := msauth{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return "", fmt.Errorf("getXSTStoken: %w", err)
	}
	return m.Token, nil
}

func loginWithXbox(uhs string, xstsToken string) (string, error) {
	msg := `{"identityToken": "XBL3.0 x=` + jsonEscape(uhs) + `;` + jsonEscape(xstsToken) + `"}`
	b, err := httPost(loginWithXboxURL, msg, `application/json`)
	if err != nil {
		return "", fmt.Errorf("loginWithXbox: %w", err)
	}
	t := token{}
	err = json.Unmarshal(b, &t)
	if err != nil {
		return "", fmt.Errorf("loginWithXbox: %w", err)
	}
	return t.AccessToken, nil
}

func GetProfile(Authorization string) (*Profile, error) {
	reqs, err := http.NewRequest("GET", getTheprofileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("getProfile: %w", err)
	}
	reqs.Header.Set("Authorization", "Bearer "+Authorization)
	rep, err := c.Do(reqs)
	if rep != nil {
		defer rep.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("getProfile: %w", err)
	}
	b, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return nil, fmt.Errorf("getProfile: %w", err)
	}
	p := Profile{
		AccessToken: Authorization,
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		return nil, fmt.Errorf("getProfile: %w", err)
	}
	if p.ID == "" {
		return nil, ErrProfile
	}
	return &p, nil
}

type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string
}

type msauth struct {
	DisplayClaims displayClaims `json:"DisplayClaims"`
	IssueInstant  string        `json:"IssueInstant"`
	NotAfter      string        `json:"NotAfter"`
	Token         string        `json:"Token"`
}

type displayClaims struct {
	Xui []xui `json:"xui"`
}

type xui struct {
	Uhs string `json:"uhs"`
}
type token struct {
	AccessToken string `json:"access_token"`
}

var (
	ErrCode    = errors.New("code invalid")
	ErrToken   = errors.New("Token invalid")
	ErrProfile = errors.New("DO NOT HAVE GAME")
)

func httPost(url, msg, ContentType string) ([]byte, error) {
	reqs, err := http.NewRequest("POST", url, strings.NewReader(msg))
	if err != nil {
		return nil, fmt.Errorf("httPost: %w", err)
	}
	reqs.Header.Set("Content-Type", ContentType)
	reqs.Header.Set("Accept", "*/*")
	reqs.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")
	rep, err := c.Do(reqs)
	if rep != nil {
		defer rep.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("httPost: %w", err)
	}
	b, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return nil, fmt.Errorf("httPost: %w", err)
	}
	return b, nil
}

var c = &http.Client{
	Timeout:   15 * time.Second,
	Transport: Transport,
}

func jsonEscape(s string) string {
	b, err := json.Marshal(&s)
	if err != nil {
		panic(err)
	}
	r := []rune(string(b))
	if len(r) == 0 {
		return ""
	}
	if r[0] == '"' {
		r = r[1:]
	}
	if r[len(r)-1] == '"' {
		r = r[:len(r)-1]
	}
	return string(r)
}
