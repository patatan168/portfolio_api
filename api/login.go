package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"encoding/json"

	"fmt"
	"os"
	"time"

	"server/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

/* Userの構造体(データベースの列) */
type User struct {
	Username   string `json:"user_name" xml:"user_name" form:"user_name"`
	Id         string `json:"id" xml:"id" form:"id"`
	Password   string `json:"password" xml:"password" form:"password"`
	Type       string `json:"type" xml:"type" form:"type"`
	Uuid       string `json:"uuid" xml:"uuid" form:"uuid"`
	Privatekey string `json:"private_key" xml:"private_key" form:"private_key"`
}

/* テーブル名 */
const userTable string = "user_list"

/* Auth Time(H) */
const authTime time.Duration = time.Hour * 72

/* todoデータベースへ接続 */
func ConnUser(app *fiber.App, uri string) {
	database := uri
	// クエリー
	userLogin(app, database)
	addUser(app, database)
	makeTestUser(app, database)
	loginTestUser(app, database)
	loginTestAuth(app, database)
}

func createTokenCookie(c *fiber.Ctx, token string) {
	// Create cookie
	cookie := new(fiber.Cookie)
	cookie.Name = "token"
	cookie.Value = token
	cookie.Expires = time.Now().Add(time.Duration(authTime))
	cookie.Path = "/"
	cookie.HTTPOnly = true
	cookie.SameSite = "Lax"
	// Set cookie
	c.Cookie(cookie)
}

func userLogin(app *fiber.App, database string) {
	app.Post(postDbRoute(userTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Login (%v)\n", userTable)
		// JSON Parse
		var user User
		if err := json.Unmarshal(c.Body(), &user); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQL Connection
		ctx, conn := connection(database)
		// Has Exists User?
		if hasExists, _ := auth.HasUserExists(ctx, conn, auth.WithUserId(user.Id), auth.WithPassword(user.Password)); !hasExists {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		errUuid := conn.QueryRow(ctx, "select uuid from "+userTable+" where id = '"+auth.CreateHexSha3(user.Id)+"' and password = '"+auth.CreateHexSha3(user.Password)+"'").Scan(&user.Uuid)
		if errUuid != nil {
			conn.Close(ctx)
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// Create Private Key
		conn.QueryRow(ctx, "select private_key from "+userTable+" where uuid = '"+user.Uuid+"'").Scan(&user.Privatekey)
		privateKey := auth.CreatePrivateKey(user.Privatekey)
		// Create the Claims
		claims := auth.CreateClaims(c, user.Uuid, authTime)
		// Create token
		tmpToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
		// Generate encoded token and send it as response
		token, errToken := tmpToken.SignedString(privateKey)
		if errToken != nil {
			conn.Close(ctx)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Create cookie
		createTokenCookie(c, token)

		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "OK\n")
		return c.SendStatus(fiber.StatusOK)
	})
}

func addUser(app *fiber.App, database string) {
	app.Get(getDbRoute(userTable)+"/add", func(c *fiber.Ctx) error {
		// JSON Parse
		var user User
		if err := json.Unmarshal(c.Body(), &user); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// Create Private Key
		privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		// PgSQLへデータを追加
		ctx, conn := connection(database)
		// UIDの生成
		uuid, _ := uuid.NewRandom()
		// ハッシュ化
		shaId := auth.CreateHexSha3(user.Id)
		shaPassword := auth.CreateHexSha3(user.Password)
		_, err := conn.Exec(ctx, `
		insert into user_list (user_name, id, password, type, uuid, private_key) values
			('`+user.Username+`', '`+shaId+`', '`+shaPassword+`', '`+user.Type+`', '`+uuid.String()+`', `+privateKey.D.String()+`);
		`)
		defer conn.Close(ctx)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendStatus(fiber.StatusOK)
	})
}

func makeTestUser(app *fiber.App, database string) {
	app.Get(getDbRoute(userTable), func(c *fiber.Ctx) error {
		// PgSQLへデータを追加
		ctx, conn := connection(database)
		id := "test"
		password := "test"
		userType := "test"
		name := "test"
		// ハッシュ化
		shaId := auth.CreateHexSha3(id)
		shaPassword := auth.CreateHexSha3(password)

		privateKey, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if keyErr != nil {
			fmt.Fprintf(os.Stderr, "Key %v\n", keyErr)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// UIDの生成
		uuid, _ := uuid.NewRandom()
		_, err := conn.Exec(ctx, `
		insert into user_list (user_name, id, password, type, uuid, private_key) values
			('`+name+`', '`+shaId+`', '`+shaPassword+`', '`+userType+`', '`+uuid.String()+`', `+privateKey.D.String()+`);
		`)
		defer conn.Close(ctx)
		if err == nil {
			return c.Status(fiber.StatusOK).SendString(privateKey.D.String())
		} else {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	})
}

func loginTestUser(app *fiber.App, database string) {
	app.Get("/user/test", func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Login (%v)\n", userTable)
		// JSON Parse
		var user User
		user.Id = "test"
		user.Password = "test"

		// PgSQL Connection
		ctx, conn := connection(database)
		// Has Exists User?
		if hasExists, errLogin := auth.HasUserExists(ctx, conn, auth.WithUserId(user.Id), auth.WithPassword(user.Password)); !hasExists || errLogin != nil {
			conn.Close(ctx)
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		errUuid := conn.QueryRow(ctx, "select uuid from "+userTable+" where id = '"+auth.CreateHexSha3(user.Id)+"' and password = '"+auth.CreateHexSha3(user.Password)+"'").Scan(&user.Uuid)
		if errUuid != nil {
			conn.Close(ctx)
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// Create Private Key
		conn.QueryRow(ctx, "select private_key from "+userTable+" where uuid = '"+user.Uuid+"'").Scan(&user.Privatekey)
		privateKey := auth.CreatePrivateKey(user.Privatekey)
		// Create the Claims
		claims := auth.CreateClaims(c, user.Uuid, authTime)
		// Create token
		token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
		// Generate encoded token and send it as response.
		t, errToken := token.SignedString(privateKey)
		if errToken != nil {
			fmt.Fprintf(os.Stderr, "ErrorToken %v\n", errToken)
			conn.Close(ctx)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Create cookie
		cookie := new(fiber.Cookie)
		cookie.Name = "token"
		cookie.Value = t
		cookie.Expires = time.Now().Add(time.Duration(authTime))
		cookie.Path = "/"
		cookie.HTTPOnly = true
		cookie.SameSite = "Lax"

		// Set cookie
		c.Cookie(cookie)

		defer conn.Close(ctx)
		return c.SendStatus(fiber.StatusOK)
	})
}
func loginTestAuth(app *fiber.App, database string) {
	app.Get("/user/test2", func(c *fiber.Ctx) error {
		tst, tst2, test := auth.VerifyToken(c, database)
		fmt.Fprintf(os.Stderr, "%v\n%v\n%v\n", tst, tst2, test)

		return c.SendStatus(fiber.StatusOK)
	})
}
