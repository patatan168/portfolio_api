package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/xid"
)

/* Todoの構造体(データベースの列) */
type Todo struct {
	Id   string    `json:"id" xml:"id" form:"id"`
	Todo string    `json:"todo" xml:"todo" form:"todo"`
	Time time.Time `json:"time" xml:"time" form:"time"`
}

/* テーブル名 */
const todoTable string = "todo"

/* Get分割(文字列) */
const pageDiv string = "10"

/* Get分割(uInt) */
func pageDivInt() uint64 {
	tmp, _ := strconv.ParseUint(pageDiv, 10, 8)
	return tmp
}

/* Todoデータベースへ接続 */
func ConnTodo(app *fiber.App, uri string) {
	database := uri

	// クエリー
	todoGet(app, database)
	todoGetDiv(app, database)
	todoPost(app, database)
	todoPut(app, database)
	todoDelete(app, database)
}

func todoGet(app *fiber.App, database string) {
	app.Get(getDbRoute(todoTable), func(c *fiber.Ctx) error {
		// データベースからTodoを取得していく(日付が新しい順)
		ctx, conn := connection(database)
		results, queryErr := conn.Query(ctx, "select * from "+todoTable+" order by time desc"+" limit "+pageDiv)
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", todoTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		todos := make([]Todo, 0, pageDivInt())
		tmp := new(Todo)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Id, &tmp.Todo, &tmp.Time); err != nil {
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

func todoGetDiv(app *fiber.App, database string) {
	app.Get(getDbRoute(todoTable)+"/:lastTime", func(c *fiber.Ctx) error {
		// データベースからTodoを取得していく(日付が新しい順)
		ctx, conn := connection(database)
		lastTime := c.Params("lastTime")
		results, queryErr := conn.Query(ctx, "select * from "+todoTable+" where time < '"+lastTime+"' order by time desc limit "+pageDiv)
		defer conn.Close(ctx)
		fmt.Fprintf(os.Stderr, "Get (%v)\n", todoTable)
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Query %v\n", queryErr)
			return c.SendStatus(500)
		}
		// JSONデータの配列を作成
		todos := make([]Todo, 0, pageDivInt())
		tmp := new(Todo)
		for results.Next() {
			// 一時変数にオブジェクトを格納
			if err := results.Scan(&tmp.Id, &tmp.Todo, &tmp.Time); err != nil {
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

func todoPost(app *fiber.App, database string) {
	app.Post(postDbRoute(todoTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Post (%v)\n", todoTable)
		// 送信されたJSONをパース
		var todo Todo
		if err := json.Unmarshal(c.Body(), &todo); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLへデータを追加
		ctx, conn := connection(database)
		// UIDの生成
		uid := xid.New()
		// Timestampの生成(UTC:RFC3339Nano)
		timeStamp := time.Now().UTC().Format(time.RFC3339Nano)
		_, err := conn.Exec(ctx, `
		insert into `+todoTable+` (id, todo, time) values
			('`+uid.String()+`', '`+todo.Todo+`', '`+timeStamp+`');
		`)
		defer conn.Close(ctx)
		if err == nil {
			return c.Status(fiber.StatusOK).SendString(todo.Todo)
		} else {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	})
}

func todoPut(app *fiber.App, database string) {
	app.Put(putDbRoute(todoTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Put (%v)\n", todoTable)
		// 送信されたJSONをパース
		var todo Todo
		if err := json.Unmarshal(c.Body(), &todo); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLへデータを追加
		ctx, conn := connection(database)
		_, err := conn.Query(ctx, "update "+todoTable+" set todo='"+todo.Todo+"' where id='"+todo.Id+"'")
		defer conn.Close(ctx)
		if err == nil {
			return c.Status(fiber.StatusOK).SendString(todo.Todo)
		} else {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	})
}

func todoDelete(app *fiber.App, database string) {
	app.Delete(deleteDbRoute(todoTable), func(c *fiber.Ctx) error {
		fmt.Fprintf(os.Stderr, "Delete (%v)\n", todoTable)
		// 送信されたJSONをパース
		var todo Todo
		if err := json.Unmarshal(c.Body(), &todo); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// PgSQLのデータを削除
		ctx, conn := connection(database)
		_, err := conn.Query(ctx, deleteDbReq(todoTable, "id", todo.Id))
		defer conn.Close(ctx)
		if err == nil {
			return c.Status(fiber.StatusOK).SendString(todo.Todo)
		} else {
			fmt.Fprintf(os.Stderr, "Query %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
	})
}
