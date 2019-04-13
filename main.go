package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func getUpdatesChan(bot *tgbotapi.BotAPI) tgbotapi.UpdatesChannel {
	// Webhook, if available
	if WebhookURL != `` && WebhookPattern != ``{
		if info, err := bot.GetWebhookInfo(); err == nil {
			if info.IsSet() {
				_, _ = bot.RemoveWebhook()
			}
			if resp, err := bot.SetWebhook(tgbotapi.NewWebhook(WebhookURL)); err != nil {
				log.Fatalln(err)
			} else if !resp.Ok {
				log.Fatalln(resp)
			} else {
				return bot.ListenForWebhook(WebhookPattern)
			}
		} else {
			log.Fatalln(err)
		}
	}

	// Longpool
	if updates, err := bot.GetUpdatesChan(tgbotapi.UpdateConfig{}); err == nil {
		return updates
	} else {
		log.Fatalln(err)
	}
	return nil
}

func updateFullName(db *redis.Client, user *tgbotapi.User) {
	if err := db.Set(strconv.Itoa(user.ID) + `-fullName`, getFullName(user), 0).Err(); err != nil {
		log.Println(`updateFullName`, strconv.Itoa(user.ID), getFullName(user), err)
	}
}

func requestFullName(db *redis.Client, userID int) string {
	if name, err := db.Get(strconv.Itoa(userID) + `-fullName`).Result(); err == nil {
		return name
	} else {
		log.Println(`requestFullName`, strconv.Itoa(userID), err)
	}
	return `<???>`
}

func register(db *redis.Client, chatID int64, userID int) error {
	return db.SAdd(strconv.FormatInt(chatID, 10) + `-members`, userID).Err()
}

func unregister(db *redis.Client, chatID int64, userID int) error {
	return db.SRem(strconv.FormatInt(chatID, 10) + `-members`, userID).Err()
}

func peekWinners(db *redis.Client, chatID int64) (int, int, error) {
	var result []int
	err := db.SRandMemberN(strconv.FormatInt(chatID, 10) + `-members`, 2).ScanSlice(&result)
	if err == nil {
		if len(result) == 2 {
			return result[0], result[1], nil
		} else if len(result) == 1 {
			return result[0], result[0], nil
		}
	}
	return 0, 0, err
}

type Results []int

func (s Results) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s Results) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &s)
}

func saveResults(db *redis.Client, chatID int64, faggotID, niceID int) {
	data, _ := json.Marshal([]int{faggotID, niceID})
	if err := db.Set(strconv.FormatInt(chatID, 10) + `-result`, data, time.Minute).Err(); err != nil {
		log.Println(err)
	}
}

func getResults(db *redis.Client, chatID int64) (int, int, error) {
	if data, err := db.Get(strconv.FormatInt(chatID, 10) + `-result`).Bytes(); err == nil {
		var result []int
		if err := json.Unmarshal(data, &result); err == nil {
			return result[0], result[1], nil
		} else {
			return 0, 0, err
		}
	} else {
		return 0, 0, err
	}
}

func getFullName(user *tgbotapi.User) string {
	name := user.FirstName
	if user.LastName != `` {
		name += ` ` + user.LastName
	}
	return name
}

func getMentionUser(user *tgbotapi.User) string {
	// Make inline mention of a user
	// https://core.telegram.org/bots/api#markdown-style
	return getMention(getFullName(user), user.ID)
}

func getMention(fullName string, userID int) string {
	// Make inline mention of a user
	// https://core.telegram.org/bots/api#markdown-style
	return fmt.Sprintf(`[%s](tg://user?id=%d)`, fullName, userID)
}

func newMessage(chatID int64, text string) tgbotapi.MessageConfig {
	message := tgbotapi.NewMessage(chatID, text)
	message.ParseMode = tgbotapi.ModeMarkdown
	return message
}

func processCommand(lang Language, db *redis.Client, bot *tgbotapi.BotAPI, update tgbotapi.Update)  {
	switch update.Message.Command() {
	case `reg`:
		if err := register(db, update.Message.Chat.ID, update.Message.From.ID); err == nil {
			text := fmt.Sprintf(lang.Registered, getMentionUser(update.Message.From))
			if _, err := bot.Send(newMessage(update.Message.Chat.ID, text)); err != nil {
				log.Println(err)
			}
		} else {
			if _, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, lang.ErrorOccurred)); err != nil {
				log.Println(err)
			}
			log.Println(err)
		}
		break
	case `ignore`:
		if err := unregister(db, update.Message.Chat.ID, update.Message.From.ID); err == nil {
			text := fmt.Sprintf(lang.Unregistered, getMentionUser(update.Message.From))
			if _, err := bot.Send(newMessage(update.Message.Chat.ID, text)); err != nil {
				log.Println(err)
			}
		} else {
			if _, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, lang.ErrorOccurred)); err != nil {
				log.Println(err)
			}
			log.Println(err)
		}
		break
	case `run`:
		if faggotUserID, niceUserID, err := getResults(db, update.Message.Chat.ID); err == nil {
			faggotMessageText := fmt.Sprintf(lang.RunFaggot, requestFullName(db, faggotUserID))
			niceMessageText := fmt.Sprintf(lang.RunNice, requestFullName(db, niceUserID))
			if _, err := bot.Send(newMessage(update.Message.Chat.ID, strings.Join([]string{faggotMessageText, niceMessageText}, "\n"))); err != nil {
				log.Println(err)
			}
		} else if err == redis.Nil {
			if faggotUserID, niceUserID, err := peekWinners(db, update.Message.Chat.ID); err == nil {
				faggotMessageText := fmt.Sprintf(lang.RunFaggot, getMention(requestFullName(db, faggotUserID), faggotUserID))
				niceMessageText := fmt.Sprintf(lang.RunNice, getMention(requestFullName(db, niceUserID), niceUserID))
				if _, err := bot.Send(newMessage(update.Message.Chat.ID, strings.Join([]string{faggotMessageText, niceMessageText}, "\n"))); err != nil {
					log.Println(err)
				}
				saveResults(db, update.Message.Chat.ID, faggotUserID, niceUserID)
			} else {
				if _, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, lang.ErrorOccurred)); err != nil {
					log.Println(err)
				}
				log.Println(err)
			}
		} else {
			if _, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, lang.ErrorOccurred)); err != nil {
				log.Println(err)
			}
			log.Println(err)
		}
		break
	}
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.LUTC)
	log.SetOutput(os.Stdout)

	go func() {
		panic(http.ListenAndServe(`:` + PORT, nil))
	}()

	if opts, err := redis.ParseURL(RedisURL); err == nil {
		db := redis.NewClient(opts)
		if bot, err := tgbotapi.NewBotAPI(BotToken); err == nil {
			bot.Debug = DEBUG
			lang := Languages[`ru`] // TODO add language choice
			for update := range getUpdatesChan(bot) {
				if update.Message != nil {
					updateFullName(db, update.Message.From)
					if update.Message.IsCommand() && update.Message.Chat.IsSuperGroup() || update.Message.Chat.IsGroup() {
						go processCommand(lang, db, bot, update)
					}
				}
			}
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Fatalln(err)
	}

}
