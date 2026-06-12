package ctxkey

import "context"

type key string

const UserID key = "userID"

func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(UserID).(string)
	return id
}
