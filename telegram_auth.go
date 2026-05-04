package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type telegramInitDataUser struct {
	ID int64 `json:"id"`
}

type telegramInitDataChat struct {
	ID int64 `json:"id"`
}

func validateTelegramInitData(initData string, botToken string) (TelegramAuthContext, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return TelegramAuthContext{}, fmt.Errorf("parse init data: %w", err)
	}

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return TelegramAuthContext{}, fmt.Errorf("missing hash")
	}
	values.Del("hash")

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	dataCheckString := strings.Join(parts, "\n")

	secretHash := hmac.New(sha256.New, []byte("WebAppData"))
	secretHash.Write([]byte(botToken))
	secret := secretHash.Sum(nil)

	dataHash := hmac.New(sha256.New, secret)
	dataHash.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(dataHash.Sum(nil))

	if !hmac.Equal([]byte(computedHash), []byte(receivedHash)) {
		return TelegramAuthContext{}, fmt.Errorf("invalid hash")
	}

	var user telegramInitDataUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID <= 0 {
		return TelegramAuthContext{}, fmt.Errorf("invalid user")
	}

	chatID := user.ID
	if chatValue := values.Get("chat"); chatValue != "" {
		var chat telegramInitDataChat
		if err := json.Unmarshal([]byte(chatValue), &chat); err != nil || chat.ID == 0 {
			return TelegramAuthContext{}, fmt.Errorf("invalid chat")
		}
		chatID = chat.ID
	}

	return TelegramAuthContext{ChatID: chatID, UserID: user.ID}, nil
}
