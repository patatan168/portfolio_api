package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"strings"

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

/* User取得用の構造体 */
type UserGet struct {
	Username string `json:"user_name" xml:"user_name" form:"user_name"`
	Uuid     string `json:"uuid" xml:"uuid" form:"uuid"`
	Type     string `json:"type" xml:"type" form:"type"`
}

type HasLogin struct {
	Valid bool `json:"hasLogin" xml:"hasLogin" form:"hasLogin"`
}

/* テーブル名 */
const userTable string = "user_list"

/* Auth Time(H) */
const authTime time.Duration = time.Duration(72) * time.Hour

/* todoデータベースへ接続 */
func ConnUser(app *fiber.App, uri string) {
	database := uri
	// クエリー
	hasLogin(app, database)
	userLogin(app, database)
	addUser(app, database)
	userGet(app, database)
	userDelete(app, database)
}

func createTokenCookie(c *fiber.Ctx, token string) {
	// Create cookie
	cookie := new(fiber.Cookie)
	cookie.Name = "token"
	cookie.Value = token
	cookie.Expires = time.Now().Add(authTime)
	cookie.HTTPOnly = true
	cookie.Secure = true

	// Set SameSite
	hostName := c.Hostname()
	isDev := strings.Contains(hostName, "127.0.0.1") || strings.Contains(hostName, "localhost")
	// 開発環境ではNoneにしとく
	if isDev {
		cookie.SameSite = "None"
	} else {
		cookie.SameSite = "Lax"
	}
	// Set cookie
	c.Cookie(cookie)
}

func hasLogin(app *fiber.App, database string) {
	app.Get(getDbRoute(userTable)+"/haslogin", func(c *fiber.Ctx) error {
		valid, _, err := auth.VerifyToken(c, database)
		var hasLogin HasLogin
		hasLogin.Valid = valid
		hostName := c.Hostname()
		fmt.Fprintf(os.Stderr, "%v\n", hostName)
		if valid {
			c.JSON(hasLogin)
			fmt.Fprintf(os.Stderr, "Has Login (%v)\n", userTable)
			return c.SendStatus(fiber.StatusOK)
		} else {
			c.JSON(hasLogin)
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	})
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
	app.Post(postDbRoute(userTable)+"/add", func(c *fiber.Ctx) error {
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		if !valid || authType != auth.TypeMap[auth.Admin] {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
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
		insert into `+userTable+` (user_name, id, password, type, uuid, private_key) values
			('`+user.Username+`', '`+shaId+`', '`+shaPassword+`', '`+user.Type+`', '`+uuid.String()+`', `+privateKey.D.String()+`);
		`)
		defer conn.Close(ctx)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendStatus(fiber.StatusOK)
	})
}

func userGet(app *fiber.App, database string) {
	app.Get(getDbRoute(userTable), func(c *fiber.Ctx) error {
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		if !valid || authType != auth.TypeMap[auth.Admin] {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// データベースから取得
		ctx, conn := connection(database)
		results, queryErr := conn.Query(ctx, "select user_name, uuid, type from "+userTable+" order by type")
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", userTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		todos := make([]UserGet, 0)
		tmp := new(UserGet)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Username, &tmp.Uuid, &tmp.Type); err != nil {
				fmt.Fprintf(os.Stderr, "Rows Next: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			// オブジェクトの配列を追加
			todos = append(todos, *tmp)
		}
		// JSONデータを出力
		c.JSON(todos)
		return c.SendStatus(fiber.StatusOK)
	})
}

func userDelete(app *fiber.App, database string) {
	app.Delete(deleteDbRoute(userTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Delete (%v)\n", userTable)
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		if !valid || authType != auth.TypeMap[auth.Admin] {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// 送信されたJSONをパース
		var user User
		if err := json.Unmarshal(c.Body(), &user); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLのデータを削除
		ctx, conn := connection(database)
		_, err := conn.Query(ctx, deleteDbReq(userTable, "uuid", user.Uuid))
		defer conn.Close(ctx)
		if err == nil {
			return c.Status(fiber.StatusOK).SendString(user.Uuid)
		} else {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	})
}
