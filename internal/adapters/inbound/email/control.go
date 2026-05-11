package email

import "github.com/webitel/flow_manager/model"

func (c *connection) Reply(text string) (*model.Email, *model.AppError) {
	var email *model.Email
	p, err := c.GetProfile()
	if err != nil {
		return nil, err
	}
	email, err = p.Reply(c.email, []byte(text))

	if err != nil {
		return nil, err
	}

	return email, nil
}
