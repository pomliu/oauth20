package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	templateoauth "github.com/ldrogou/goauth20/templateOAuth"
)

type JsonToken struct {
	clientID     string `json:"client_id"`
	clientSecret string `json:"client_secret"`
	grantType    string `json:"grant_type"`
	redirectURI  string `json:"redirect_uri"`
	code         string `json:"code"`
}

type token struct {
	accessToken  string `json:"access_token"`
	tokenType    string `json:"token_type"`
	expiresIn    int    `json:"expires_in"`
	refreshToken string `json:"refresh_token"`
}

// Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	Sub          string   `json:"sub"`
	IDEntreprise string   `json:"idEntreprise"`
	RcaPartnerID string   `json:"rcaPartnerId"`
	Roles        []string `json:"roles"`
	jwt.StandardClaims
}

func (s *server) handleIndex() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)

		t, err := template.New("test").Parse(templateoauth.TemplateIndex)
		if err != nil {
			fmt.Errorf("erreur suivante %v", err)
		}

		err = t.Execute(rw, nil)
		if err != nil {
			fmt.Errorf("erreur suivante %v", err)
		}
	}

}
func (s *server) handleLocal() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		sub := r.FormValue("sub")
		idEntreprise := r.FormValue("id_entreprise")
		rcaPartnerID := r.FormValue("rcaPartnerId")
		var jwtKey = []byte(r.FormValue("secret"))

		// Declare the expiration time of the token
		// here, we have kept it as 5 minutes
		expirationTime := time.Now().Add(5 * time.Hour)
		roles := []string{"RCA_CLOUD_EXPERT_COMPTABLE",
			"E_COLLECTE_BO_CREA",
			"E_CREATION_CREA",
			"E_QUESTIONNAIRE_CREA"}
		// Create the JWT claims, which includes the username and expiry time
		claims := &Claims{
			Sub:          sub,
			IDEntreprise: idEntreprise,
			RcaPartnerID: rcaPartnerID,
			Roles:        roles,
			StandardClaims: jwt.StandardClaims{
				// In JWT, the expiry time is expressed as unix milliseconds
				ExpiresAt: expirationTime.Unix(),
			},
		}

		// Declare the token with the algorithm used for signing, and the claims
		tokenstr := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Create the JWT string
		tokenString, err := tokenstr.SignedString(jwtKey)
		if err != nil {
			log.Printf("erreur %v", err)
			// If there is an error in creating the JWT return an internal server error
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		s.responseFile(rw, r, tokenString, http.StatusOK)
	}

}

func (s *server) handleOAuth20() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		d := r.FormValue("domain")
		ci := r.FormValue("clientId")
		s := r.FormValue("scopes")
		cc := r.FormValue("currentCompany")
		if len(cc) == 0 {
			cc = "false"
		} else {
			cc = "true"
		}

		rhttp := "https://" + d + "/entreprise-partenaire/authorize?client_id=" + ci +
			"&scope=" + s +
			"&current_company=" + cc +
			"&redirect_uri=http://localhost:8080/oauth/redirect" +
			"&abort_uri=http://localhost:8080/index"
		http.Redirect(rw, r, rhttp, http.StatusMovedPermanently)

	}

}

func (s *server) handleRedirect() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		codes, _ := r.URL.Query()["code"]
		jsonStr := constJsonToken(codes[0])

		apiURL := "https://api.captation.beta.rca.fr/auth/v1/oauth2.0/accessToken"
		data := url.Values{}
		data.Set("client_id", jsonStr.clientID)
		data.Set("client_secret", jsonStr.clientSecret)
		data.Set("grant_type", jsonStr.grantType)
		data.Set("redirect_uri", jsonStr.redirectURI)
		data.Set("code", jsonStr.code)

		client := &http.Client{}
		req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(data.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
		req.Header.Add("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		var t map[string]interface{}
		// here's the trick
		json.NewDecoder(resp.Body).Decode(&t)

		if err != nil {
			log.Printf("Cannot parse token body err=%v", err)
			s.response(rw, r, nil, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if err != nil {
			log.Printf("Cannot parse token body err=%v", err)
			s.response(rw, r, nil, http.StatusBadGateway)
			return
		}

		s.responseFile(rw, r, t["access_token"], http.StatusOK)

	}
}

func constJsonToken(code string) JsonToken {
	return JsonToken{
		clientID:     "meg-test-interne",
		clientSecret: "YNVZF88dD4vny59k",
		grantType:    "authorization_code",
		redirectURI:  "http://localhost:8080/callback",
		code:         code,
	}
}
