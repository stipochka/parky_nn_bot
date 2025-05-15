package handler

import (
	"log/slog"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	greetingText = `Здравствуйте!
Выберете раздел:`
	ntoFilePath = "/Users/stipochka/course_tg_bot/files/sales/SHABLON.doc"
	meroChoice  = "Вы выбрали: Мероприятия"
	saleChoice  = "Вы выбрали: Торговля/услуги"
	attrChoice  = "Вы выбрали: Развлечения"
	salesPhotos = "/Users/stipochka/course_tg_bot/files/sales/photo/"
	attrPhotos  = "/Users/stipochka/course_tg_bot/files/attr/photo/"
	docMessage  = `Заполните шаблон заявки и скан или фото заполненного и подписанного документа пришлите на почту info@parkinnov.ru. Поставьте печать, если работаете с печатью. После рассмотрения с вами свяжутся для заключения договора.`
)

var log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

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

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	lgr := log.With(slog.String("user", message.Chat.UserName))
	switch message.Text {
	case "/start":
		lgr.Info("received /start command")
		menu := tgbotapi.NewMessage(message.Chat.ID, greetingText)
		menu.ReplyMarkup = MainMenu()
		bot.Send(menu)
		lgr.Info("sended menu to user")
	case "📅 Мероприятия":
		lgr.Info("called events menu")
		text := tgbotapi.NewMessage(message.Chat.ID, meroChoice)
		bot.Send(text)
		sendInlineMenu(bot, message.Chat.ID, eventsMenu())
		lgr.Info("sended menu to user")
	case "📦 Торговля/услуги":
		lgr.Info("called trade/services menu")
		text := tgbotapi.NewMessage(message.Chat.ID, saleChoice)
		bot.Send(text)
		sendInlineMenu(bot, message.Chat.ID, servicesMenu())
		lgr.Info("sended trade/services menu")
	case "🎡 Развлечения":
		lgr.Info("called attraction menu")
		text := tgbotapi.NewMessage(message.Chat.ID, attrChoice)
		bot.Send(text)
		sendInlineMenu(bot, message.Chat.ID, attractionsMenu())
		lgr.Info("sended attraction menu")
	}
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
		log.Info("sended price list to user")
	case "application_form":
		if _, err := os.Stat(ntoFilePath); err == nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, docMessage)
			doc := tgbotapi.NewDocument(query.Message.Chat.ID, tgbotapi.FilePath(ntoFilePath))
			bot.Send(doc)
			bot.Send(msg)
			log.Info("sended application from to user")
			return
		}
		log.Error("failed to send application form to user")

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
