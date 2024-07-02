package auth

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

type token struct {
	Token string `cookie:"token"`
}

type user struct {
	Username   string `json:"user_name" xml:"user_name" form:"user_name"`
	Id         string `json:"id" xml:"id" form:"id"`
	Password   string `json:"password" xml:"password" form:"password"`
	Type       string `json:"type" xml:"type" form:"type"`
	Uuid       string `json:"uuid" xml:"uuid" form:"uuid"`
	Privatekey string `json:"private_key" xml:"private_key" form:"private_key"`
}

func VerifyToken(c *fiber.Ctx, database string) (bool, string, error) {
	cookie, errCookie := fetchToken(c)
	if errCookie != nil {
		return false, "", errCookie
	}
	// JSON Parse
	var user user
	// Get Token
	token, _ := jwt.Parse(cookie.Token, func(token *jwt.Token) (interface{}, error) {
		// Not Verify
		return []byte(""), nil
	})
	claims, errClaims := token.Claims.(jwt.MapClaims)
	if !errClaims {
		return false, "", errors.New("claims error")
	}
	// Get Audience IP
	aud, _ := claims.GetAudience()
	var reqIP jwt.ClaimStrings = []string{c.IP()}
	if !reflect.DeepEqual(aud, reqIP) {
		return false, "", fmt.Errorf("not verify [aud] / [reqIP] %v / %v", aud, reqIP)
	}
	// Get UUID
	user.Uuid, _ = claims.GetSubject()
	// Access to SQL
	ctx := context.Background()
	conn, _ := pgx.Connect(ctx, database)
	// Has Exits User?
	hasExists, errExists := HasUserExists(ctx, conn, WithUuid(user.Uuid))
	if !hasExists || errExists != nil {
		conn.Close(ctx)
		return false, "", fmt.Errorf("not exists user")
	}
	// Fetch
	conn.QueryRow(ctx, "select type, private_key from "+userTable+" where uuid = '"+user.Uuid+"'").Scan(&user.Type, &user.Privatekey)
	// Create Private Key
	publicKey := CreatePrivateKey(user.Privatekey).PublicKey
	// Parse JWT
	_, valid := jwt.Parse(cookie.Token, func(token *jwt.Token) (interface{}, error) {
		// Verify PublicKey
		return &publicKey, nil
	})
	if valid != nil {
		return false, "", fmt.Errorf("not valid %v", valid)
	}
	conn.Close(ctx)
	return true, user.Type, nil
}

func fetchToken(c *fiber.Ctx) (*token, error) {
	cookie := new(token)
	if err := c.CookieParser(cookie); err != nil {
		return nil, err
	}
	if cookie.Token == "" {
		return nil, errors.New("cookie undefined")
	}
	return cookie, nil
}
