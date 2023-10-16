package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"message-bot-demo/controllers"
	"message-bot-demo/db"
	"message-bot-demo/flows"
	"message-bot-demo/utils"
	"net/http"
	"net/url"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/gorilla/mux"
)

// for telegram integration: send a telegram message equivalent to the provided MessageNode
// to the specified user
func SendMessage(bot *tgbotapi.BotAPI, chatId string, messageNode *flows.MessageNode) (*tgbotapi.Message, error) {
	intChatId, err := strconv.ParseInt(chatId, 10, 64)

	if err != nil {
		return nil, err
	}

	haveButtons := false
	var keyboard tgbotapi.ReplyKeyboardMarkup

	if len(messageNode.Button) > 0 {
		haveButtons = true
		keyboardButtons := utils.Map(messageNode.Button,
			func(t string) tgbotapi.KeyboardButton {
				return tgbotapi.NewKeyboardButton(t)
			},
		)

		keyboard = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				keyboardButtons...,
			),
		)
	}

	var message tgbotapi.Chattable

	if messageNode.Image != "" {
		parsedUrl, err := url.Parse(messageNode.Image)
		if err == nil {
			// We have a valid image url
			photoMsg := tgbotapi.NewPhotoUpload(intChatId, parsedUrl)

			if haveButtons {
				photoMsg.ReplyMarkup = keyboard
			}

			if messageNode.Body != "" {
				photoMsg.Caption = messageNode.Body
			}
			message = photoMsg
		}
	} else {
		msg := tgbotapi.NewMessage(intChatId, messageNode.Body)

		if haveButtons {
			msg.ReplyMarkup = keyboard
		}
	}

	sendMsg, err := bot.Send(message)

	if err != nil {
		return nil, err
	}

	return &sendMsg, nil
}

// TelegramInteraction not used, currently stdin/stdout is used
// for interaction with the bot
func TelegramInteraction(bot *tgbotapi.BotAPI) {

	updates := bot.ListenForWebhook("/" + bot.Token)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		messageText := update.Message.Text
		username := update.Message.From.UserName

		nextNode, err := controllers.HandleMessageActiveFlow(username, messageText, false)

		if err != nil {
		}

		message := tgbotapi.NewMessage(update.Message.Chat.ID, nextNode.Body)

		log.Printf("%v\n", update.Message)
		log.Printf("%v\n", update.Message.From)
		_, err = bot.Send(message)

		if err != nil {
			log.Println(err.Error())

		}
	}
}

// TerminalInteraction For testing purposes use stdin/stdout for interaction
func TerminalInteraction(username string) {

	updatesFromStdin := make(chan string)

	go func(ch chan string) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		close(ch)
	}(updatesFromStdin)

	for messageText := range updatesFromStdin {

		nextNode, err := controllers.HandleMessageActiveFlow(username, messageText, false)

		if err != nil {
			// Use other method for non-deterministic processing
			// Either ML responses or hand-off to human agent
			log.Println(err.Error())
		}

		if nextNode != nil {
			fmt.Println(nextNode.ToString())
		}
	}
}

func main() {

	username := flag.String("username", "test", "username for cli interaction")
	flag.Parse()

	const dbpath string = "demo.db"

	// initialize db for testing purposes
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		err = db.InitTables(dbpath)

		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	err := db.Open(dbpath)
	_ = db.PopulateDB()
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer db.Close()

	// display the representation of a flow for debugging purposes
	demoFlow := flows.DemoFlowFactory()
	fmt.Println(demoFlow.ToString())

	// interact with the bot using stdin/stdout
	// TODO use a seperate program
	go TerminalInteraction(*username)

	// api for initiating conversations and displaying stats
	// TODO possibly create a more fine-grained api to handle
	// message replies
	router := mux.NewRouter()
	router.HandleFunc("/api/messages", controllers.MessageHandler)
	router.HandleFunc("/api/stats", controllers.StatsHandler)

	const port int = 3000
	log.Printf("Starting server on port %d...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}
