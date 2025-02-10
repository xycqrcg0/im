package utils

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"im/global"
	"math/rand"
	"time"
)

//func CreateID() string {
//	return uuid.New().String()
//}

func HashPassword(pwd string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	return string(hashed), err
}

func CheckPassword(hash string, pwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
}

func GenerateJWT(name string, id string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": name,
		"id":       id,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	}) //此时token包含头和载荷
	signedToken, err := token.SignedString([]byte("cookie")) //此时完成最后签名

	return "Bearer " + signedToken, err
}

func ParseJWT(tokenString string) (string, string, error) {
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("token 方法不符")
		}
		return []byte("cookie"), nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid { //claims是token的有效载荷部分
		name, ok1 := claims["username"].(string)
		id, ok2 := claims["id"].(string)
		if !ok1 {
			return "", "", errors.New("name is not a string")
		}
		if !ok2 {
			return "", "", errors.New("id is not a string")
		}
		return name, id, nil
	}

	return "", "", err
}

func GenerateUserID() string {
	var id string
	for {
		intId := rand.Intn(1000000) + 1
		id = fmt.Sprintf("%06d", intId)
		if _, ok := global.HaveUsedID[id]; !ok {
			break
		}
	}
	return id
}
