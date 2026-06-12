package model

type Interactive struct {
	Documents *Documents `json:"documents,omitempty"`
	Images    *Images    `json:"images,omitempty"`

	SingleUse bool `json:"singleUse"`

	Markup    *KeyboardMarkup    `json:"markup,omitempty"`
	ListReply *KeyboardListReply `json:"listReply,omitempty"`
}

type Documents struct {
	Documents []File `json:"documents"`
}

type Images struct {
	Images []File `json:"images"`
}

type KeyboardRow struct {
	Buttons []KeyboardButton `json:"buttons"`
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

type KeyboardRowGeneric[B any] struct {
	Buttons []B `json:"buttons"`
}

type KeyboardMarkupGeneric[B any] struct {
	Rows []KeyboardRowGeneric[B] `json:"rows"`
}

type KeyboardRowWithSectionGeneric[B any] struct {
	Section string `json:"section"`
	Buttons []B    `json:"buttons"`
}

type KeyboardListReplyGeneric[B any] struct {
	MainButtonTitle string                             `json:"mainButtonTitle"`
	Sections        []KeyboardRowWithSectionGeneric[B] `json:"sections"`
}

type KeyboardMarkup = KeyboardMarkupGeneric[KeyboardButton]
type KeyboardListReply = KeyboardListReplyGeneric[KeyboardButton]

type InteractiveGeneric[B any] struct {
	Documents *Documents                   `json:"documents,omitempty"`
	Images    *Images                      `json:"images,omitempty"`
	SingleUse bool                         `json:"singleUse"`
	Markup    *KeyboardMarkupGeneric[B]    `json:"markup,omitempty"`
	ListReply *KeyboardListReplyGeneric[B] `json:"listReply,omitempty"`
}

type SendInteractiveRequestGeneric[B any] struct {
	Interactive InteractiveGeneric[B] `json:"interactive"`
	Body        string                `json:"body"`
	Metadata    map[string]any        `json:"metadata"`
}

type SendInteractiveRequest = SendInteractiveRequestGeneric[KeyboardButton]
