package email

import (
	"errors"
	"io"
	"net/http"

	"github.com/emersion/go-message/mail"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

type MailPartReader interface {
	NextPart() (*mail.Part, error)
}

type MailIterator struct {
	reader MailPartReader
	logger *wlog.Logger
	curr   *mail.Part
	err    *model.AppError
}

func NewMailIterator(reader MailPartReader, logger *wlog.Logger) *MailIterator {
	return &MailIterator{reader: reader, logger: logger}
}

func (it *MailIterator) Next() bool {
	if it.err != nil {
		return false
	}

	part, err := it.reader.NextPart()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			it.logger.Error("getting next part from mail reader", wlog.Err(err))
			it.err = model.NewAppError("Next", "email.reader_iterator.next_part", nil, err.Error(), http.StatusInternalServerError)
		}

		it.curr = nil

		return false
	}

	if part == nil {
		it.logger.Warn("empty part")
		it.curr = nil

		return false
	}

	it.curr = part

	return true
}

func (it *MailIterator) Part() *mail.Part     { return it.curr }
func (it *MailIterator) Err() *model.AppError { return it.err }
