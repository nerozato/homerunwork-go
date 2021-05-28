package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//chat db tables
const (
	dbTableChat    = "chat"
	dbTableChatMsg = "chat_message"
)

//Chat : definition of a chat
type Chat struct {
	ID         *uuid.UUID `json:"-"`
	ProviderID *uuid.UUID `json:"-"`
	OrderID    *uuid.UUID `json:"-"`
	Email      string     `json:"-"`
}

//ChatMsg : definition of a chat
type ChatMsg struct {
	ID      *uuid.UUID `json:"-"`
	ChatID  *uuid.UUID `json:"-"`
	Created time.Time  `json:"-"`
	Text    string     `json:"Text"`
}

//save a chat
func saveChat(ctx context.Context, db *DB, chat *Chat) (context.Context, error) {
	//generate an id if necessary
	if chat.ID == nil {
		chatID, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid")
		}
		chat.ID = &chatID
	}

	//json encode the chat data
	chatJSON, err := json.Marshal(chat)
	if err != nil {
		return ctx, errors.Wrap(err, "json message")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,order_id,email,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?)", dbTableChat)
	ctx, result, err := db.Exec(ctx, stmt, chat.ID, chat.ProviderID, chat.OrderID, chat.Email, chatJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert chat")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert chat rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert chat: %s: %s", chat.ProviderID, chat.Email)
	}
	return ctx, nil
}

//load a chat
func loadChat(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *Chat, error) {
	stmt := fmt.Sprintf("SELECT id,provider_id,order_id,email,data FROM %s WHERE %s", dbTableChat, whereStmt)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row chat")
	}

	//read the row
	var idStr string
	var providerIDStr string
	var orderIDStr string
	var email string
	var dataStr string
	err = row.Scan(&idStr, &providerIDStr, &orderIDStr, &email, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "select chat")
	}

	//parse the id
	id, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid id")
	}
	providerID, err := uuid.FromString(providerIDStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid provider id")
	}
	orderID, err := uuid.FromString(orderIDStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid order id")
	}

	//unmarshal the data
	var chat Chat
	err = json.Unmarshal([]byte(dataStr), &chat)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson chat")
	}
	chat.ID = &id
	chat.ProviderID = &providerID
	chat.OrderID = &orderID
	chat.Email = email
	return ctx, &chat, nil
}

//load a chat by order id and email
func loadChatByOrderIDAndEmail(ctx context.Context, db *DB, orderID *uuid.UUID, email string) (context.Context, *Chat, error) {
	whereStmt := "deleted=0 AND order_id=UUID_TO_BIN(?) AND email=?"
	ctx, chat, err := loadChat(ctx, db, whereStmt, orderID, email)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "load chat")
	}
	return ctx, chat, nil
}

//load a chat by provider id and email
func loadChatByProviderIDAndEmail(ctx context.Context, db *DB, providerID *uuid.UUID, email string) (context.Context, *Chat, error) {
	whereStmt := "deleted=0 AND provider_id=UUID_TO_BIN(?) AND email=?"
	ctx, chat, err := loadChat(ctx, db, whereStmt, providerID, email)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "load chat")
	}
	return ctx, chat, nil
}

//ListChatsByProviderID : list the chats by provider
func ListChatsByProviderID(ctx context.Context, db *DB, providerID *uuid.UUID) (context.Context, []*Chat, error) {
	ctx, logger := GetLogger(ctx)

	//list the chats
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),BIN_TO_UUID(order_id),email,data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?)", dbTableChat)
	ctx, rows, err := db.Query(ctx, stmt, providerID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list chats")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//create the chats
	var idStr string
	var orderIDStr string
	var email string
	var dataStr string
	chats := make([]*Chat, 0, 5)
	for rows.Next() {
		err := rows.Scan(&idStr, &orderIDStr, &email, &dataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan chats")
		}

		//parse the id
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid id")
		}
		orderID, err := uuid.FromString(orderIDStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid order id")
		}

		//unmarshal the data
		var chat Chat
		err = json.Unmarshal([]byte(dataStr), &chat)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson chat")
		}
		chat.ID = &id
		chat.ProviderID = providerID
		chat.OrderID = &orderID
		chat.Email = email
		chats = append(chats, &chat)
	}
	return ctx, chats, nil
}

//save a chat message
func saveChatMsg(ctx context.Context, db *DB, chatMsg *ChatMsg) (context.Context, error) {
	//generate an id if necessary
	if chatMsg.ID == nil {
		chatMsgID, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid")
		}
		chatMsg.ID = &chatMsgID
	}

	//json encode the chat data
	chatMsgJSON, err := json.Marshal(chatMsg)
	if err != nil {
		return ctx, errors.Wrap(err, "json chat message")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,chat_id,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?)", dbTableChatMsg)
	ctx, result, err := db.Exec(ctx, stmt, chatMsg.ID, chatMsg.ChatID, chatMsgJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert chat message")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert chat message rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert chat message: %s", chatMsg.ChatID)
	}
	return ctx, nil
}

//SaveChatMsgForOrderIDAndEmail : save a chat message baseed on the order id
func SaveChatMsgForOrderIDAndEmail(ctx context.Context, db *DB, chatMsg *ChatMsg, providerID *uuid.UUID, orderID *uuid.UUID, email string) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save chat message provider", func(ctx context.Context, db *DB) (context.Context, error) {
		//check for the chat and create if necessary
		ctx, chat, err := loadChatByOrderIDAndEmail(ctx, db, orderID, email)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load chat message order: %s: %s", orderID, email))
		}
		if chat == nil {
			chat = &Chat{
				ProviderID: providerID,
				OrderID:    orderID,
				Email:      email,
			}
			ctx, err = saveChat(ctx, db, chat)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("save chat order: %s: %s", orderID, email))
			}
		}
		chatMsg.ChatID = chat.ID

		//save the message
		ctx, err = saveChatMsg(ctx, db, chatMsg)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("save chat message order: %s: %s", orderID, email))
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save chat message order")
	}
	return ctx, nil
}

//SaveChatMsgForProviderIDAndEmail : save a chat message baseed on the provider id
func SaveChatMsgForProviderIDAndEmail(ctx context.Context, db *DB, chatMsg *ChatMsg, providerID *uuid.UUID, email string) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save chat message provider", func(ctx context.Context, db *DB) (context.Context, error) {
		//check for the chat and create if necessary
		ctx, chat, err := loadChatByProviderIDAndEmail(ctx, db, providerID, email)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load chat message provider: %s: %s", providerID, email))
		}
		if chat == nil {
			chat = &Chat{
				ProviderID: providerID,
				Email:      email,
			}
			ctx, err = saveChat(ctx, db, chat)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("save chat provider: %s: %s", providerID, email))
			}
		}
		chatMsg.ChatID = chat.ID

		//save the message
		ctx, err = saveChatMsg(ctx, db, chatMsg)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("save chat message provider: %s: %s", providerID, email))
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save chat message provider")
	}
	return ctx, nil
}

//list all chat messages
func listChatMsgs(ctx context.Context, db *DB, prev string, next string, limit int, whereStmt string, args ...interface{}) (context.Context, []*ChatMsg, string, string, error) {
	ctx, logger := GetLogger(ctx)
	const delimiter = "-"
	var prevStr string
	var nextStr string
	var stmt string
	dbArgs := make([]interface{}, 0, 3)

	//make the appropriate query based on if paginating
	if prev != "" {
		//sort descending to walk backwards, though the results should be reversed
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(cm.id),cm.data,cm.created FROM %s cm INNER JOIN %s c ON c.id=cm.chat_id AND c.deleted=0 WHERE cm.deleted=0 AND (cm.created<? OR (cm.created=? AND cm.id<UUID_TO_BIN(?))) AND %s ORDER BY cm.created DESC,cm.id DESC LIMIT %d", dbTableChatMsg, dbTableChat, whereStmt, limit)
		tokens := strings.Split(prev, delimiter)
		time := ParseTimeUnixUTC(tokens[0])
		id := DecodeUUIDBase64(tokens[1])
		dbArgs = []interface{}{time, time, id}
	} else if next != "" {
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(cm.id),cm.data,cm.created FROM %s cm INNER JOIN %s c ON c.id=cm.chat_id AND c.deleted=0 WHERE cm.deleted=0 AND (cm.created>? OR (cm.created=? AND cm.id>UUID_TO_BIN(?))) AND %s ORDER BY cm.created,cm.id LIMIT %d", dbTableChatMsg, dbTableChat, whereStmt, limit)
		tokens := strings.Split(next, delimiter)
		time := ParseTimeUnixUTC(tokens[0])
		id := DecodeUUIDBase64(tokens[1])
		dbArgs = []interface{}{time, time, id}
	} else {
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(cm.id),cm.data,cm.created FROM %s cm INNER JOIN %s c ON c.id=cm.chat_id AND c.deleted=0 WHERE cm.deleted=0 AND %s ORDER BY cm.created,cm.id LIMIT %d", dbTableChatMsg, dbTableChat, whereStmt, limit)
	}

	//incorporate the additional arguments
	dbArgs = append(dbArgs, args...)
	ctx, rows, err := db.Query(ctx, stmt, dbArgs...)
	if err != nil {
		return ctx, nil, "", "", errors.Wrap(err, "select chats")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//bookkeeping for previous and next pages
	var idFirst *uuid.UUID
	var idLast *uuid.UUID
	var keyFirst time.Time
	var keyLast time.Time

	//read the rows
	chatMsgs := make([]*ChatMsg, 0, 2)
	var idStr string
	var chatIDStr string
	var dataStr string
	var created time.Time
	for rows.Next() {
		err := rows.Scan(&idStr, &chatIDStr, &dataStr, &created)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "rows scan chat messages")
		}

		//parse the id
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "parse uuid id")
		}
		chatID, err := uuid.FromString(chatIDStr)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "parse uuid chat id")
		}

		//unmarshal the data
		var chatMsg ChatMsg
		err = json.Unmarshal([]byte(dataStr), &chatMsg)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "unjson chat message")
		}
		chatMsg.ID = &id
		chatMsg.ChatID = &chatID
		chatMsg.Created = created
		chatMsgs = append(chatMsgs, &chatMsg)
	}

	//reverse the list if going to a previous page, since the sort was descending
	lenChatMsgs := len(chatMsgs)
	if prev != "" {
		for i, j := 0, lenChatMsgs-1; i < j; i, j = i+1, j-1 {
			chatMsgs[i], chatMsgs[j] = chatMsgs[j], chatMsgs[i]
		}
	}

	//determine the first and last entries to determine the previous and next links
	if lenChatMsgs > 0 {
		idFirst = chatMsgs[0].ID
		idLast = chatMsgs[lenChatMsgs-1].ID
		keyFirst = chatMsgs[0].Created
		keyLast = chatMsgs[lenChatMsgs-1].Created
	}

	//check if there's a previous page
	if idFirst != nil && !keyFirst.IsZero() {
		stmt = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND (created<? OR (created=? AND p.id<UUID_TO_BIN(?))) ORDER BY created DESC,id DESC LIMIT %d", dbTableChatMsg, limit)
		dbArgs = []interface{}{keyFirst, keyFirst, idFirst}
		ctx, row, err := db.QueryRow(ctx, stmt, dbArgs...)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "query row providers count prev")
		}
		var count int
		err = row.Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			return ctx, nil, "", "", errors.Wrap(err, "select providers count prev")
		}
		if count > 0 {
			prevStr = fmt.Sprintf("%d%s%s", keyFirst.UnixNano(), delimiter, EncodeUUIDBase64(idFirst))
		}
	}

	//check if there's a next page
	if idLast != nil && !keyLast.IsZero() {
		stmt = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND (created>? OR (created=? AND p.id>UUID_TO_BIN(?))) ORDER BY created,id LIMIT %d", dbTableChatMsg, limit)
		dbArgs = []interface{}{keyLast, keyLast, idLast}
		ctx, row, err := db.QueryRow(ctx, stmt, dbArgs...)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "query row providers count next")
		}
		var count int
		err = row.Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			return ctx, nil, "", "", errors.Wrap(err, "select providers count next")
		}
		if count > 0 {
			nextStr = fmt.Sprintf("%d%s%s", keyLast.Unix(), delimiter, EncodeUUIDBase64(idLast))
		}
	}
	return ctx, chatMsgs, prevStr, nextStr, nil
}

//ListChatMsgsByChatID : list chat messages by chat id
func ListChatMsgsByChatID(ctx context.Context, db *DB, prev string, next string, limit int, chatID uuid.UUID) (context.Context, []*ChatMsg, string, string, error) {
	whereStmt := "c.id=UUID_TO_BIN(?)"
	ctx, msgs, prevStr, nextStr, err := listChatMsgs(ctx, db, prev, next, limit, whereStmt, chatID)
	if err != nil {
		return ctx, nil, "", "", err
	}
	return ctx, msgs, prevStr, nextStr, nil
}

//ListChatMsgsByOrderID : list chat messages by order id
func ListChatMsgsByOrderID(ctx context.Context, db *DB, prev string, next string, limit int, orderID uuid.UUID) (context.Context, []*ChatMsg, string, string, error) {
	whereStmt := "c.order_id=UUID_TO_BIN(?)"
	ctx, msgs, prevStr, nextStr, err := listChatMsgs(ctx, db, prev, next, limit, whereStmt, orderID)
	if err != nil {
		return ctx, nil, "", "", err
	}
	return ctx, msgs, prevStr, nextStr, nil
}

//ListChatMsgsByProviderIDAndEmail : list chat messages by provider id and email
func ListChatMsgsByProviderIDAndEmail(ctx context.Context, db *DB, prev string, next string, limit int, providerID uuid.UUID, email string) (context.Context, []*ChatMsg, string, string, error) {
	whereStmt := "c.provider_id=UUID_TO_BIN(?) AND c.email=?"
	ctx, msgs, prevStr, nextStr, err := listChatMsgs(ctx, db, prev, next, limit, whereStmt, providerID, email)
	if err != nil {
		return ctx, nil, "", "", err
	}
	return ctx, msgs, prevStr, nextStr, nil
}
