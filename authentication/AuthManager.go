package authentication

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"
)

// Create the JWT key used to create the signature
var jwtKey = []byte("")

var users = map[string]string{
	"user1": "password1",
	"user2": "password2",
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type Token struct {
	Token string `json:"token"`
	Success bool `json:"success"`
}

type LoginFailed struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

func Signin(w http.ResponseWriter, r *http.Request) {
	var credential Credentials
	err := json.NewDecoder(r.Body).Decode(&credential)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pass, ok := users[credential.Username]
	if !ok || bcrypt.CompareHashAndPassword([]byte(pass), []byte(credential.Password)) != nil {
		log.Printf("[INFO][%s] Login failed for user %s\n", r.RemoteAddr, credential.Username)
		SendError(w, r, "Credentials doesn't matches")
		return
	}
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: credential.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("[INFO][%s] %s\n", r.RemoteAddr, err)
		SendError(w, r, err.Error())
		return
	}

	res:= Token{
		Token: tokenString,
		Success: true,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	} else {
		log.Printf("[INFO][%s] Login successful for user %s\n", r.RemoteAddr, credential.Username)
	}
}

func SendError(w http.ResponseWriter, r *http.Request, msg string) {
	res:= LoginFailed{
		Message: msg,
		Success: false,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Add("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	}
}