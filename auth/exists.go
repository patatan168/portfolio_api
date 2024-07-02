package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const userTable string = "user_list"

type exitsAuthOptions struct {
	uuid     string
	id       string
	password string
}

type optionExists func(*exitsAuthOptions)

func HasUserExists(ctx context.Context, conn *pgx.Conn, options ...optionExists) (bool, error) {
	opt := exitsAuthOptions{}
	argNum := 0
	for _, o := range options {
		argNum++
		o(&opt)
	}
	if argNum == 1 && opt.uuid != "" {
		return hasUserExistsUuidAuth(ctx, conn, opt.uuid)
	}
	if argNum == 2 && opt.id != "" && opt.password != "" {
		return hasUserExistsIdPassAuth(ctx, conn, opt.id, opt.password)
	}
	return false, errors.New("function error")
}
func WithUuid(uuid string) optionExists {
	return func(ops *exitsAuthOptions) {
		ops.uuid = uuid
	}
}
func WithUserId(id string) optionExists {
	return func(ops *exitsAuthOptions) {
		ops.id = id
	}
}
func WithPassword(password string) optionExists {
	return func(ops *exitsAuthOptions) {
		ops.password = password
	}
}

func hasUserExistsUuidAuth(ctx context.Context, conn *pgx.Conn, uuid string) (bool, error) {
	var hasExists bool
	errLogin := conn.QueryRow(ctx, "select exists (select * from "+userTable+" where uuid = '"+uuid+"')").Scan(&hasExists)

	return hasExists, errLogin
}
func hasUserExistsIdPassAuth(ctx context.Context, conn *pgx.Conn, id string, password string) (bool, error) {
	var hasExists bool
	// ハッシュ化
	shaId := CreateHexSha3(id)
	shaPassword := CreateHexSha3(password)
	errLogin := conn.QueryRow(ctx, "select exists (select * from "+userTable+" where id = '"+shaId+"' and password = '"+shaPassword+"')").Scan(&hasExists)

	return hasExists, errLogin
}
