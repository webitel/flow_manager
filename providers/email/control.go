package email

import "github.com/webitel/flow_manager/model"

func (c *connection) Reply(text string) (model.Response, *model.AppError) {
	err := c.profile.Reply(c.email, []byte(text))

	if err != nil {
		return model.CallResponseError, err
	}

	return model.CallResponseOK, nil
}
