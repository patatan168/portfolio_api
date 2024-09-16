package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"server/auth"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

/* Blogの構造体(データベースの列) */
type Blog struct {
	Path     string    `json:"path" xml:"path" form:"path"`
	Tag      string    `json:"tag" xml:"tag" form:"tag"`
	Title    string    `json:"title" xml:"title" form:"title"`
	Uuid     string    `json:"uuid" xml:"uuid" form:"uuid"`
	Sentence string    `json:"sentence" xml:"sentence" form:"sentence"`
	EditTime time.Time `json:"edit_time" xml:"edit_time" form:"edit_time"`
	PostTime time.Time `json:"post_time" xml:"post_time" form:"post_time"`
}

/* テーブル名 */
const blogTable string = "blog"

const entryList string = "path, tag, title, uuid, post_time"
const editList string = "uuid, path, title, edit_time, post_time"
const entry string = "path, tag, title, uuid, sentence, edit_time, post_time"
const putEntry string = "tag, title, sentence, edit_time"

/* Get分割(文字列) */
const blogPageDiv string = "5"

/* Get分割(uInt) */
func blogPageDivInt() uint64 {
	tmp, _ := strconv.ParseUint(blogPageDiv, 10, 8)
	return tmp
}

type existsGetEntryListOpt struct {
	lastTime string
	path     string
}

type optionGetEntryList func(*existsGetEntryListOpt)

/* DBからEntryの情報を取得 */
func getEntryListData(ctx context.Context, conn *pgx.Conn, options ...optionGetEntryList) (pgx.Rows, error) {
	opt := existsGetEntryListOpt{}
	argNum := 0
	for _, o := range options {
		argNum++
		o(&opt)
	}
	if argNum == 0 {
		return conn.Query(ctx, "select "+entryList+" from "+blogTable+" order by post_time desc"+" limit "+blogPageDiv)
	}
	if argNum == 1 && opt.lastTime != "" {
		return conn.Query(ctx, "select "+entryList+" from "+blogTable+" where post_time < '"+opt.lastTime+"' order by post_time desc limit "+blogPageDiv)
	}
	if argNum == 1 && opt.path != "" {
		return conn.Query(ctx, "select "+entry+" from "+blogTable+" where path = '"+opt.path+"'")
	}
	return nil, errors.New("getEntryListData error")
}

func withLastTime(lastTime string) optionGetEntryList {
	return func(ops *existsGetEntryListOpt) {
		ops.lastTime = lastTime
	}
}

func withPath(path string) optionGetEntryList {
	return func(ops *existsGetEntryListOpt) {
		ops.path = path
	}
}

/* Blogデータベースへ接続 */
func ConnBlog(app *fiber.App, uri string) {
	database := uri

	// クエリー
	blogGet(app, database)
	blogGetDiv(app, database)
	blogEditGet(app, database)
	blogGetEntry(app, database)
	blogPost(app, database)
	blogPut(app, database)
	blogDelete(app, database)
}

func blogGet(app *fiber.App, database string) {
	app.Get(getDbRoute(blogTable), func(c *fiber.Ctx) error {
		// データベースからBlogを取得していく(日付が新しい順)
		ctx, conn := connection(database)
		results, queryErr := getEntryListData(ctx, conn)
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", blogTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		blogs := make([]Blog, 0, blogPageDivInt())
		tmp := new(Blog)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Path, &tmp.Tag, &tmp.Title, &tmp.Uuid, &tmp.PostTime); err != nil {
				fmt.Fprintf(os.Stderr, "Rows Next: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			// オブジェクトの配列を追加
			blogs = append(blogs, *tmp)
		}
		// JSONデータを出力
		c.JSON(blogs)
		return c.SendStatus(fiber.StatusOK)
	})
}

func blogGetDiv(app *fiber.App, database string) {
	app.Get(getDbRoute(blogTable)+"/:lastTime", func(c *fiber.Ctx) error {
		// データベースからBlogを取得していく(日付が新しい順)
		ctx, conn := connection(database)
		lastTime := c.Params("lastTime")
		results, queryErr := getEntryListData(ctx, conn, withLastTime(lastTime))
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", blogTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		blogs := make([]Blog, 0, blogPageDivInt())
		tmp := new(Blog)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Path, &tmp.Tag, &tmp.Title, &tmp.Uuid, &tmp.PostTime); err != nil {
				fmt.Fprintf(os.Stderr, "Rows Next: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			// オブジェクトの配列を追加
			blogs = append(blogs, *tmp)
		}
		// JSONデータを出力
		c.JSON(blogs)
		return c.SendStatus(fiber.StatusOK)
	})
}

/* 編集用のブログListを出力 */
func blogEditGet(app *fiber.App, database string) {
	app.Get(getDbRoute(blogTable)+"-edit", func(c *fiber.Ctx) error {
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		// Admin or Blogger
		auth := authType != auth.TypeMap[auth.Admin] && authType != auth.TypeMap[auth.Bloger]
		if !valid || auth {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// データベースからBlogを取得していく(日付が新しい順)
		ctx, conn := connection(database)
		results, queryErr := conn.Query(ctx, "select "+editList+" from "+blogTable)
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", blogTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		var blogs []Blog
		tmp := new(Blog)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Uuid, &tmp.Path, &tmp.Title, &tmp.EditTime, &tmp.PostTime); err != nil {
				fmt.Fprintf(os.Stderr, "Rows Next: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			// オブジェクトの配列を追加
			blogs = append(blogs, *tmp)
		}
		// JSONデータを出力
		c.JSON(blogs)
		return c.SendStatus(fiber.StatusOK)
	})
}

func blogGetEntry(app *fiber.App, database string) {
	app.Get(getDbRoute(blogTable)+"-entry"+"/:path", func(c *fiber.Ctx) error {
		// データベースからBlogを取得していく(日付が新しい順)
		ctx, conn := connection(database)
		path := c.Params("path")
		results, queryErr := getEntryListData(ctx, conn, withPath(path))
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", blogTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		blogs := make([]Blog, 0, 1)
		tmp := new(Blog)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Path, &tmp.Tag, &tmp.Title, &tmp.Uuid, &tmp.Sentence, &tmp.EditTime, &tmp.PostTime); err != nil {
				fmt.Fprintf(os.Stderr, "Rows Next: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			// オブジェクトの配列を追加
			blogs = append(blogs, *tmp)
		}
		// JSONデータを出力
		c.JSON(blogs)
		return c.SendStatus(fiber.StatusOK)
	})
}

func blogPost(app *fiber.App, database string) {
	app.Post(postDbRoute(blogTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Post (%v)\n", blogTable)
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		if !valid || authType != auth.TypeMap[auth.Admin] {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// 送信されたJSONをパース
		var blog Blog
		if err := json.Unmarshal(c.Body(), &blog); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLへデータを追加
		ctx, conn := connection(database)
		// pathが被ってないか確認する
		var hasExists bool
		conn.QueryRow(ctx, "select exists (select * from "+blogTable+" where path = '"+blog.Path+"')").Scan(&hasExists)
		if hasExists {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		// UIDの生成
		uuid, _ := uuid.NewRandom()
		// Timestampの生成(UTC:RFC3339Nano)
		timeStamp := time.Now().UTC().Format(time.RFC3339Nano)
		nullTime := time.Unix(0, 0).Format(time.RFC3339Nano)
		// Post
		_, err := conn.Exec(ctx, `
		insert into `+blogTable+` (`+entry+`) values
			('`+blog.Path+`', '`+blog.Tag+`', '`+blog.Title+`', '`+uuid.String()+`', '`+blog.Sentence+`', '`+nullTime+`', '`+timeStamp+`');
		`)
		defer conn.Close(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendStatus(fiber.StatusOK)
	})
}

func blogPut(app *fiber.App, database string) {
	app.Put(putDbRoute(blogTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Put (%v)\n", blogTable)
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		if !valid || authType != auth.TypeMap[auth.Admin] {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// 送信されたJSONをパース
		var blog Blog
		if err := json.Unmarshal(c.Body(), &blog); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLへデータを追加
		ctx, conn := connection(database)
		// Timestampの生成(UTC:RFC3339Nano)
		timeStamp := time.Now().UTC().Format(time.RFC3339Nano)
		// Put
		_, err := conn.Query(ctx, `
		update `+blogTable+`
			set (`+putEntry+`) = (`+blog.Tag+`, `+blog.Sentence+`, `+blog.Sentence+`, `+timeStamp+`)
			where uuid='`+blog.Uuid+`'
		`)
		defer conn.Close(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendStatus(fiber.StatusOK)
	})
}

func blogDelete(app *fiber.App, database string) {
	app.Delete(deleteDbRoute(blogTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Delete (%v)\n", blogTable)
		// Verify Token
		valid, authType, _ := auth.VerifyToken(c, database)
		if !valid || authType != auth.TypeMap[auth.Admin] {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		// 送信されたJSONをパース
		var blog Blog
		if err := json.Unmarshal(c.Body(), &blog); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLのデータを削除
		ctx, conn := connection(database)
		_, err := conn.Query(ctx, deleteDbReq(blogTable, "uuid", blog.Uuid))
		defer conn.Close(ctx)
		if err == nil {
			return c.SendStatus(fiber.StatusOK)
		} else {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	})
}
