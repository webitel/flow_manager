package email

import "github.com/webitel/flow_manager/model"

func (c *connection) Reply(text string) (*model.Email, *model.AppError) {
	email, err := c.profile.Reply(c.email, []byte(text))

	if err != nil {
		return nil, err
	}

	return email, nil
}
