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
var cost = 12
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

type UserInfo struct {
	Username string `json:"username"`
	Administrator bool `json:"administrator"`
	Success bool `json:"success"`
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

type EditedUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Administrator bool `json:"administrator"`
	PasswordEdited bool `json:"password-edited"`
	AdministratorEdited bool `json:"administrator-edited"`
}

func LoadUsers() {
	if _, err := os.Stat(ConfigurationManager.GetConfiguration().UsersList); err != nil {
		log.Fatalf("[ERROR] The users file is not found ::> users.json\n%s", err)
	}
	configJson, err:= ioutil.ReadFile(ConfigurationManager.GetConfiguration().UsersList)
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
	t, err := GetToken(r)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	loggedUser, err := GetLoggedUser(t)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	if loggedUser.Administrator {
		var credential User
		err = json.NewDecoder(r.Body).Decode(&credential)
		if err != nil {
			SendError(w, r, "User information missing (username and/or password)")
			return
		}
		_, err = GetUserInfo(credential.Username)
		if err == nil {
			SendError(w, r, "User already exist")
			return
		}
		pass, err:= bcrypt.GenerateFromPassword([]byte(credential.Password), cost)
		if err != nil {
			SendError(w, r, err.Error())
			return
		}
		user:= User{
			Username: credential.Username,
			Password: string(pass),
			Administrator: credential.Administrator,
		}
		users = append(users, user)
		err = SaveUsersJsonFile()
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			SendError(w, r, err.Error())
			return
		}
		log.Printf("[INFO] User %s registered\n", credential.Username)
		SendSuccess(w, r, "User registered!")
	} else {
		log.Printf("[ERROR] Missing right for %s to register a user\n", loggedUser.Username)
		SendError(w, r, "You are not allowed to do that")
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	t, err := GetToken(r)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	loggedUser, err := GetLoggedUser(t)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	var editedUser EditedUser
	err = json.NewDecoder(r.Body).Decode(&editedUser)
	if err != nil {
		SendError(w, r, "User information missing (username and/or password)")
		return
	}
	if editedUser.Username == loggedUser.Username || loggedUser.Administrator {
		index:= SliceIndex(len(users), func(i int) bool { return users[i].Username == editedUser.Username })
		if editedUser.PasswordEdited {
			pass, err:= bcrypt.GenerateFromPassword([]byte(editedUser.Password), cost)
			if err != nil {
				SendError(w, r, err.Error())
				return
			}
			users[index].Password = string(pass)
		}
		if editedUser.AdministratorEdited {
			users[index].Administrator = editedUser.Administrator
		}
		err = SaveUsersJsonFile()
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			SendError(w, r, err.Error())
			return
		}
		log.Printf("[INFO] User %s updated\n", editedUser.Username)
		SendSuccess(w, r, "User updated!")
	} else {
		log.Printf("[ERROR] Missing right for %s to update a user\n", loggedUser.Username)
		SendError(w, r, "You are not allowed to do that")
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
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return false
	}
	return t.Valid
}

func GetLoggedUser(t string) (User, error) {
	var c Claims
	token, err := jwt.ParseWithClaims(t, &c, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return User{}, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		u, err := GetUserInfo(claims.Username)
		if err != nil {
			return User{}, err
		}
		return u, nil
	}
	return User{}, errors.New("failed to get logged user")
}

func GetToken(r *http.Request) (string, error) {
	header:= r.Header.Get("authorization")
	if len(header) > 7 {
		t:= header[7:]
		return t, nil
	}
	return "", errors.New("no token found")
}

func GetLoggedUserHandler(w http.ResponseWriter, r *http.Request) {
	var c Claims
	t, err := GetToken(r)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	token, err := jwt.ParseWithClaims(t, &c, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		u, err := GetUserInfo(claims.Username)
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			SendError(w, r, err.Error())
			return
		}
		userInfo:= UserInfo{
			Username: u.Username,
			Administrator: u.Administrator,
			Success: true,
		}
		b, err := json.Marshal(userInfo)
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			SendError(w, r, err.Error())
			return
		}
		Response.SendJson(w, r, b)
	} else {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
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

func GetUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	users, ok := r.URL.Query()["u"]
	if !ok || len(users[0]) < 1 {
		SendError(w, r, "Username missing")
		return
	}
	user, err:= GetUserInfo(users[0])
	if err != nil {
		SendError(w, r, "User does not exist")
		return
	}
	userInfo:= UserInfo{
		Username: user.Username,
		Administrator: user.Administrator,
		Success: true,
	}
	b, err := json.Marshal(userInfo)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	Response.SendJson(w, r, b)
}

func RemoveUser(w http.ResponseWriter, r *http.Request) {
	t, err := GetToken(r)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	loggedUser, err := GetLoggedUser(t)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		SendError(w, r, err.Error())
		return
	}
	if loggedUser.Administrator {
		us, ok := r.URL.Query()["u"]
		if !ok || len(us[0]) < 1 {
			SendError(w, r, "Username missing")
			return
		}
		index:= SliceIndex(len(users), func(i int) bool { return users[i].Username == us[0] })
		if index == -1 {
			SendError(w, r, "User does not exist")
			return
		}
		users = RemoveIndex(users, index)
		err := SaveUsersJsonFile()
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			SendError(w, r, err.Error())
			return
		}
		log.Printf("[INFO] User %s removed\n", us[0])
		SendSuccess(w, r, "User removed!")
	} else {
		log.Printf("[ERROR] Missing right for %s to remove a user\n", loggedUser.Username)
		SendError(w, r, "You are not allowed to do that")
	}
}

func RemoveIndex(s []User, index int) []User {
	return append(s[:index], s[index+1:]...)
}

func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
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

func SaveUsersJsonFile() error {
	usersDb:= UsersJsonConfig{
		Users: users,
	}
	b, err := json.Marshal(usersDb)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("users.json", b, 0644)
	if err != nil {
		return err
	}
	return nil
}