package auth

import (
	"encoding/base64"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"time"

	"log"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

type UserToken struct {
	UserID       int64
	Role         string
	ActiveShopID int64
	ExpiredAt    int64
	jwt.StandardClaims
}

func GenerateUserAccessToken(tokenInstant *UserToken) (string, error) {
	expired := time.Now().Unix() + viper.GetInt64("auth.token_expired_time")
	tokenPassword := viper.GetString("auth.token_password")
	tokenInstant.ExpiredAt = expired
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tokenInstant)
	tokenString, err := token.SignedString([]byte(tokenPassword))

	return tokenString, err
}

func IsCorrectPassword(hashedPwd string, plainPwd string) bool {
	plainPwdBytes := []byte(plainPwd)
	byteHash := []byte(hashedPwd)

	err := bcrypt.CompareHashAndPassword(byteHash, plainPwdBytes)
	return err == nil
}

func GetUserIDFromRequest(r *http.Request) int64 {
	userID := cast.ToInt64(r.Header.Get("X-User-Id"))
	log.Printf("%v", r.Header)
	return userID
}

func GetUserRoleFromRequest(r *http.Request) string {
	role := cast.ToString(r.Header.Get("X-User-Role"))
	log.Printf("%v", r.Header)
	return role
}

func GetUserAccessToken(r *http.Request) string {
	// Grab the token from the header
	accessToken := r.Header.Get(constant.TokenKeyUserType)
	if accessToken == "" {
		accessToken = r.URL.Query().Get("access_token")
	}

	return accessToken
}

func ParseUserAccessToken(r *http.Request) (userToken *UserToken, err error) {
	userAccessToken := GetUserAccessToken(r)
	return UserAccessToken(userAccessToken)
}

func UserAccessToken(userAccessToken string) (userToken *UserToken, err error) {
	userAccessToken = strings.Replace(userAccessToken, "Bearer ", "", 1)
	if userAccessToken == "" { //Token is missing, returns with error code 401 Unauthorized
		return nil, constant.HttpErrorTokenMissing
	}

	userToken = &UserToken{}

	tokenPassword := viper.GetString("auth.token_password")
	token, err := jwt.ParseWithClaims(userAccessToken, userToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenPassword), nil
	})

	//Malformed token, returns with http code 401 as usual
	if err != nil {
		return nil, err
	}

	if !token.Valid { //Token is invalid, maybe not signed on this server
		return nil, constant.HttpErrorTokenInvalid
	}

	if userToken.ExpiredAt < time.Now().Unix() {
		return nil, constant.HttpErrorTokenExpired
	}

	return userToken, nil
}

func AuthBasic(r *http.Request) (username, apikey string) {
	token := GetUserAccessToken(r)
	if token == "" {
		return
	}

	token = strings.Replace(token, "Bearer ", "", 1)
	token = strings.Replace(token, "Basic ", "", 1)
	base, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return
	}

	auth := strings.Split(string(base), ":")
	if len(auth) != 2 {
		return
	}

	username, apikey = auth[0], token
	return
}
