package api

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

/* コネクションの確立 */
func connection(database string) (context.Context, *pgx.Conn) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
	}
	return ctx, conn
}

/* データベース「DELETE」のリクエスト */
func deleteDbReq(table string, id string) string {
	return "delete from " + table + " where id = '" + id + "'"
}

/* データベース「GET」のルーティングアドレス */
func getDbRoute(table string) string {
	return "/" + table + "/get"
}

/* データベース「POST」のルーティングアドレス */
func postDbRoute(table string) string {
	return "/" + table + "/post"
}

/* データベース「PUT」のルーティングアドレス */
func putDbRoute(table string) string {
	return "/" + table + "/put"
}

/* データベース「DELETE」のルーティングアドレス */
func deleteDbRoute(table string) string {
	return "/" + table + "/delete"
}
