package model

type SendInteractiveRequest struct {
	Interactive Interactive    `json:"interactive"`
	Body        string         `json:"body"`
	Metadata    map[string]any `json:"metadata"`
}

type Interactive struct {
	Documents *Documents `json:"documents,omitempty"`
	Images    *Images    `json:"images,omitempty"`

	SingleUse bool `json:"single_use"`

	Markup    *KeyboardMarkup    `json:"markup,omitempty"`
	ListReply *KeyboardListReply `json:"list_reply,omitempty"`
}

type Documents struct {
	Documents []File `json:"documents"`
}

type Images struct {
	Images []File `json:"images"`
}

type KeyboardMarkup struct {
	Rows []KeyboardRow `json:"rows"`
}

type KeyboardRow struct {
	Buttons []KeyboardButton `json:"buttons"`
}

type KeyboardListReply struct {
	MainButtonTitle string                   `json:"main_button_title"`
	Sections        []KeyboardRowWithSection `json:"sections"`
}

type KeyboardRowWithSection struct {
	Section string           `json:"section"`
	Buttons []KeyboardButton `json:"buttons"`
}

type KeyboardButton struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Metadata map[string]any `json:"metadata,omitempty"`

	URL      *KeyboardButtonURL      `json:"url,omitempty"`
	Callback *KeyboardButtonCallback `json:"callback,omitempty"`
	Request  *KeyboardButtonRequest  `json:"request,omitempty"`
}

type KeyboardButtonURL struct {
	URL string `json:"url"`
}

type KeyboardButtonCallback struct {
	Data string `json:"data"`
}

type KeyboardButtonRequest struct {
	Action string `json:"action"`
}
