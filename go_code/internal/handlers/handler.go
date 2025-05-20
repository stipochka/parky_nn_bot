package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	greetingText = `Здравствуйте!
Выберете раздел:`
	ntoFilePath = "/app/files/sales/SHABLON.doc"
	meroChoice  = "Вы выбрали: Мероприятия"
	saleChoice  = "Вы выбрали: Торговля/услуги"
	attrChoice  = "Вы выбрали: Развлечения"
	salesPhotos = "/app/files/sales/photo/"
	attrPhotos  = "/app/files/attr/photo/"
	docMessage  = `Заполните шаблон заявки и скан или фото заполненного и подписанного документа пришлите на почту info@parkinnov.ru. Поставьте печать, если работаете с печатью. После рассмотрения с вами свяжутся для заключения договора.`
)

var (
	userSearchResults = make(map[int64][]string)
	userSearchIndex   = make(map[int64]int)
	userStates        = make(map[int64]string)
	log               = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
)

func getAllJPGImages(dir string) ([]string, error) {
	var photos []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".jpg" {
			photos = append(photos, path)
		}
		return nil
	})
	return photos, err
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch userStates[message.Chat.ID] {
	case "awaiting_search_query":
		handleSearch(bot, message, getAllLinksFromSearch, "next_article")
	case "awaiting_search_group_query":
		handleSearch(bot, message, getAllLinksFromGroup, "next_group_article")
	default:
		handleDefault(bot, message)
	}
}

func handleSearch(bot *tgbotapi.BotAPI, message *tgbotapi.Message, searchFunc func(string) ([]string, error), nextButton string) {
	keyword := message.Text
	searchingMsg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🔍 Ищу по запросу: *%s*...", keyword))
	searchingMsg.ParseMode = "Markdown"
	sentMsg, _ := bot.Send(searchingMsg)
	links, err := searchFunc(keyword)
	deleteBotMessage(bot, message.Chat.ID, sentMsg.MessageID)

	if err != nil || len(links) == 0 {

		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Ничего не найдено."))
		log.Error("search failed", slog.Any("error", err))
		delete(userStates, message.Chat.ID)
		return
	}

	userSearchResults[message.Chat.ID] = links
	userSearchIndex[message.Chat.ID] = 0
	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("1/%d\n%s", len(links), links[0]))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Следующая", nextButton),
			tgbotapi.NewInlineKeyboardButtonData("🔙 В меню", "back"),
		),
	)
	bot.Send(msg)
	delete(userStates, message.Chat.ID)
}

func handleDefault(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	//lgr := log.With(slog.String("user", message.Chat.UserName))
	switch message.Text {
	case "/start":
		menu := tgbotapi.NewMessage(message.Chat.ID, "Выберите раздел:")
		menu.ReplyMarkup = MainMenu()
		bot.Send(menu)
	case "📅 Мероприятия":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, meroChoice))
		sendInlineMenu(bot, message.Chat.ID, eventsMenu())
	case "📦 Торговля/услуги":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, saleChoice))
		sendInlineMenu(bot, message.Chat.ID, servicesMenu())
	case "🎡 Развлечения":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, attrChoice))
		sendInlineMenu(bot, message.Chat.ID, attractionsMenu())
	case "🔍 Поиск":
		userStates[message.Chat.ID] = "awaiting_search_query"
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Введите ключевое слово для поиска:"))
	case "🔍 Поиск в группе Telegram":
		userStates[message.Chat.ID] = "awaiting_search_group_query"
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Введите ключевое слово для поиска в группе:"))
	}
}

func HandleCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery) {
	switch query.Data {
	case "price_list":
		imagePaths, err := getAllGPGImages(salesPhotos)
		if err != nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Не удалось получить документы.")
			bot.Send(msg)
			log.Error("failed to get images", slog.Any("error", err))
			return
		}
		err = sendPhotoAlbum(bot, query.Message.Chat.ID, imagePaths)
		if err != nil {
			log.Error("failed to send price list album", slog.Any("error", err.Error()))
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Не получилось отправить фото")
			bot.Send(msg)
			return
		}
		menu := tgbotapi.NewMessage(query.From.ID, "")
		menu.ReplyMarkup = MainMenu()
		bot.Send(menu)
		log.Info("sended price list to user")
	case "application_form":
		if _, err := os.Stat(ntoFilePath); err == nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, docMessage)
			doc := tgbotapi.NewDocument(query.Message.Chat.ID, tgbotapi.FilePath(ntoFilePath))
			bot.Send(doc)
			bot.Send(msg)
			log.Info("sended application from to user")
			menu := tgbotapi.NewMessage(query.From.ID, "")
			menu.ReplyMarkup = MainMenu()
			bot.Send(menu)
			return
		}
		log.Error("failed to send application form to user")
	case "next_article", "next_group_article":
		idx := userSearchIndex[query.Message.Chat.ID] + 1
		links := userSearchResults[query.Message.Chat.ID]
		if idx >= len(links) {
			ed := tgbotapi.NewEditMessageText(
				query.Message.Chat.ID, query.Message.MessageID,
				"Больше нет статей. 🔙 В меню",
			)
			ed.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
				tgbotapi.NewInlineKeyboardButtonData("🔙 В меню", "back"),
			}}}
			bot.Send(ed)
			return
		}
		userSearchIndex[query.Message.Chat.ID] = idx
		ed := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID, query.Message.MessageID,
			fmt.Sprintf("%d/%d\n%s", idx+1, len(links), links[idx]),
		)
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➡️ Следующая", query.Data),
				tgbotapi.NewInlineKeyboardButtonData("🔙 В меню", "back"),
			),
		)
		ed.ReplyMarkup = &kb
		bot.Send(ed)

	case "attr_list":
		imagePaths, err := getAllGPGImages(attrPhotos)
		if err != nil {
			log.Error("failed to get images", slog.Any("error", err))
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Не удалось получить документы.")
			bot.Send(msg)
			return
		}
		err = sendPhotoAlbum(bot, query.Message.Chat.ID, imagePaths)
		if err != nil {
			log.Error("failed to send album to user", slog.Any("error", err))
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Не получилось отправить фото")
			bot.Send(msg)
			return
		}
		menu := tgbotapi.NewMessage(query.From.ID, "")
		menu.ReplyMarkup = MainMenu()
		bot.Send(menu)
		log.Info("sended attraction album to user")
	case "back":
		deleteBotMessage(bot, query.Message.Chat.ID, query.Message.MessageID)
		menu := tgbotapi.NewMessage(query.Message.Chat.ID, "Выберите раздел:")
		menu.ReplyMarkup = MainMenu()
		bot.Send(menu)
	}
}

func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{{Text: "📅 Мероприятия"}},
		[]tgbotapi.KeyboardButton{{Text: "📦 Торговля/услуги"}},
		[]tgbotapi.KeyboardButton{{Text: "🎡 Развлечения"}},
		[]tgbotapi.KeyboardButton{{Text: "🔍 Поиск"}},
		[]tgbotapi.KeyboardButton{{Text: "🔍 Поиск в группе Telegram"}},
	)
}

func eventsMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonURL("📆 План мероприятий", "https://disk.yandex.ru/i/UwUr6rxRKLJfGw"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonURL("📸 Фотоотчеты", "https://vk.com/albums-190907367"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back"),
		},
	)
}

func servicesMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📋 Прайс-лист", "price_list"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📄 Заявка (НТО)", "application_form"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back"),
		},
	)
}

func attractionsMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("💰 Стоимость", "attr_list"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back"),
		},
	)
}

func sendInlineMenu(bot *tgbotapi.BotAPI, chatID int64, menu tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, "Выберите категорию:")
	msg.ReplyMarkup = menu
	bot.Send(msg)
}

func getAllLinksFromSearch(query string) ([]string, error) {
	if len(query) > 2 {
		query = query[:len(query)-2]
	}
	searchURL := "https://parkinnov.ru/?s=" + url.QueryEscape(query)
	resp, err := http.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	cells := doc.Find("div.wf-cell.iso-item")
	log.Info("cells count", slog.Int("count", cells.Length()))

	var links []string
	cells.Each(func(_ int, cell *goquery.Selection) {
		if a := cell.Find("a").First(); a.Length() > 0 {
			if href, ok := a.Attr("href"); ok {
				links = append(links, href)
			}
		}
	})

	return links, nil
}

func sendPhotoAlbum(bot *tgbotapi.BotAPI, chatID int64, imagePaths []string) error {
	var media []interface{}

	for _, path := range imagePaths {
		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path))
		media = append(media, photo)
	}

	group := tgbotapi.NewMediaGroup(chatID, media)
	_, err := bot.SendMediaGroup(group)
	return err
}

func getAllLinksFromGroup(keyword string) ([]string, error) {
	cmd := exec.Command("/app/venv/bin/python", "/app/search_script.py", keyword)

	links := []string{}
	out, err := cmd.Output()
	if err != nil {
		return links, err
	}

	var cmdOutput []struct {
		Date string `json:"date"`
		Link string `json:"link"`
	}

	err = json.Unmarshal(out, &cmdOutput)
	if err != nil {
		return links, err
	}

	for _, link := range cmdOutput {
		links = append(links, link.Link)
	}

	return links, nil
}

func getAllGPGImages(dir string) ([]string, error) {
	var photos []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ".jpg" {
			photos = append(photos, path)
		}

		return nil
	})

	return photos, err
}

func deleteBotMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) error {
	deleteConfig := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := bot.Request(deleteConfig)
	return err
}
