package main

import "os"

var WebhookURL = os.Getenv(`WEBHOOK_URL`)
var WebhookPattern = os.Getenv(`WEBHOOK_PATTERN`)
var BotToken = os.Getenv(`BOT_TOKEN`)

var PORT = os.Getenv(`PORT`)
var RedisURL = os.Getenv(`REDIS_URL`)
var DEBUG = os.Getenv(`DEBUG`) != ``
