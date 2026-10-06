package telegram

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	MaxButtons           = 10
	RatingCallbackPrefix = "libredesk:csat:"
)

type Button struct {
	Text string `json:"text"`
	Data string `json:"callback_data,omitempty"`
	URL  string `json:"url,omitempty"`
}

type Keyboard struct {
	Rows [][]Button `json:"inline_keyboard"`
}

func MakeKeyboard(buttons []Button) *Keyboard {
	if len(buttons) == 0 {
		return nil
	}
	rows := make([][]Button, len(buttons))
	for i, button := range buttons {
		rows[i] = []Button{button}
	}
	return &Keyboard{Rows: rows}
}

func ValidateButtons(buttons []Button) error {
	if len(buttons) > MaxButtons {
		return fmt.Errorf("too many Telegram buttons")
	}
	for _, button := range buttons {
		if strings.TrimSpace(button.Text) == "" || utf8.RuneCountInString(button.Text) > 64 {
			return fmt.Errorf("invalid Telegram button label")
		}
		if (button.Data == "") == (button.URL == "") {
			return fmt.Errorf("a Telegram button needs one action")
		}
		if len(button.Data) > 64 {
			return fmt.Errorf("Telegram callback data exceeds 64 bytes")
		}
		if button.URL != "" {
			target, err := url.Parse(button.URL)
			if err != nil || target.Host == "" || (target.Scheme != "https" && target.Scheme != "http" && target.Scheme != "tg") {
				return fmt.Errorf("invalid Telegram button URL")
			}
		}
	}
	return nil
}

func RatingButtons() []Button {
	buttons := make([]Button, 5)
	for i := range buttons {
		buttons[i] = Button{Text: fmt.Sprintf("%d ★", i+1), Data: fmt.Sprintf("%s%d", RatingCallbackPrefix, i+1)}
	}
	return buttons
}
