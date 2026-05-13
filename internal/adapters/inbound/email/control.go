package email

import emaildomain "github.com/webitel/flow_manager/internal/domain/email"

func (c *connection) Reply(text string) (*emaildomain.Email, error) {
	p, err := c.GetProfile()
	if err != nil {
		return nil, err
	}
	email, err := p.Reply(c.email, []byte(text))
	if err != nil {
		return nil, err
	}
	return email, nil
}
