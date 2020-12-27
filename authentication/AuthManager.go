package authentication

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
	"net/http"
	"openify/ConfigurationManager"
	"openify/Response"
	"os"
	"time"
)

var jwtKey = []byte("")

var users []User

type UsersJsonConfig struct {
	Users []User `json:"users"`
}

type UsersList struct {
	Users []string `json:"users"`
	Success bool `json:"success"`
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type User struct {
	Password string `json:"password"`
	Username string `json:"username"`
	Administrator bool `json:"administrator"`
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

func LoadUsers() {
	if _, err := os.Stat("users.json"); err != nil {
		log.Fatalf("[ERROR] The users file is not found ::> users.json\n%s" +
			"\nPlease insure that the file is in the same directory as the executable", err)
	}
	configJson, err:= ioutil.ReadFile("users.json")
	if err != nil {
		log.Fatalf("[ERROR] Unable to read the users file ::> users.json\n%s" +
			"\nPlease insure that the file has the reading right", err)
	}
	var uc UsersJsonConfig
	err = json.Unmarshal(configJson, &uc)

	if err != nil {
		log.Fatalf("[ERROR] Users file incorrect ::> users.json\n%s", err)
	}
	users = uc.Users
	log.Printf("[INFO] %d users loaded\n", len(users))
	jwtKey = []byte(ConfigurationManager.LoadJWTKey().Key)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var credential Credentials
	err := json.NewDecoder(r.Body).Decode(&credential)
	if err != nil {
		SendError(w, r, "User information missing (username and/or password)")
		return
	}
	user, err:= GetUserInfo(credential.Username)
	if err != nil {
		log.Printf("[INFO][%s] Login failed for user %s\n", r.RemoteAddr, credential.Username)
		SendUnauthorized(w, r)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credential.Password))
	if err != nil {
		log.Printf("[INFO][%s] Login failed for user %s\n", r.RemoteAddr, credential.Username)
		SendUnauthorized(w, r)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
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

func Register(w http.ResponseWriter, r *http.Request) {
	var credential Credentials
	err := json.NewDecoder(r.Body).Decode(&credential)
	if err != nil {
		SendError(w, r, "User information missing (username and/or password)")
		return
	}
	pass, err:= bcrypt.GenerateFromPassword([]byte(credential.Password), 12)
	if err != nil {
		SendError(w, r, err.Error())
		return
	}
	user:= User{
		Username: credential.Username,
		Password: string(pass),
	}
	users = append(users, user)
	b, err := json.Marshal(users)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	err = ioutil.WriteFile("users.json", b, 0644)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
	} else {
		log.Printf("[ERROR] User %s registered\n", credential.Username)
		SendSuccess(w, r, "User registered!")
	}
}

func GetListUsers(w http.ResponseWriter, r *http.Request) {
	var result []string
	for _, user := range users {
		result = append(result, user.Username)
	}
	ul:= UsersList{
		Users: result,
		Success: true,
	}
	b, err := json.Marshal(ul)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	Response.SendJson(w, r, b)
}

func IsLogged(token string) bool {
	_, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return false
	}
	return true
}

func GetUserInfo(username string) (User, error) {
	var result User
	for _, user := range users {
		if user.Username == username {
			result = user
		}
	}
	if result == (User{}) {
		return User{}, errors.New("user not found")
	}

	return result, nil
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

func SendSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	res:= LoginFailed{
		Message: msg,
		Success: true,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	}
}

func SendUnauthorized(w http.ResponseWriter, r *http.Request) {
	res:= LoginFailed{
		Message: "You need to login before perform this action",
		Success: false,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		return
	}
	w.WriteHeader(http.StatusUnauthorized)
	w.Header().Add("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	}
}