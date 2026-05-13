package store_adapter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"
)

var (
	mailGroup      singleflight.Group
	smtpOauthGroup singleflight.Group
)

func (a *Adapter) SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, error) {
	key := fmt.Sprintf("%d-", domainId)
	if search.Id != nil {
		key += fmt.Sprintf("%d-", *search.Id)
	}
	if search.Name != nil {
		key += fmt.Sprintf("%s-", *search.Name)
	}
	settings, err, _ := mailGroup.Do(key, func() (interface{}, error) {
		return a.store.Email().SmtpSettings(domainId, search)
	})
	if err != nil {
		return nil, err
	}
	return settings.(*model.SmtSettings), nil
}

func (a *Adapter) SmtpSettingsOAuthToken(settings *model.SmtSettings) (string, error) {
	token, err, _ := smtpOauthGroup.Do(fmt.Sprintf("%v", settings.Id), func() (interface{}, error) {
		return a.smtpOAuthToken(settings)
	})
	if err != nil {
		return "", err
	}
	return token.(string), nil
}

func (a *Adapter) smtpOAuthToken(settings *model.SmtSettings) (string, error) {
	if settings.Params == nil || settings.Params.OAuth2 == nil {
		return "", nil
	}
	oauthConfig := model.OAuthConfig(settings.Server, settings.Params.OAuth2)
	var t time.Time
	if settings.Token != nil {
		t = settings.Token.Expiry
	}
	ts := oauthConfig.TokenSource(context.Background(), settings.Token)
	newToken, err := ts.Token()
	if err != nil {
		return "", model.NewAppError("SmtSettingsOAuthToken", "app.smtp.oauth.app_err", nil, err.Error(), http.StatusInternalServerError)
	}
	if !t.Equal(newToken.Expiry) {
		if err2 := a.store.Email().SetToken(settings.Id, newToken); err2 != nil {
			wlog.Error(fmt.Sprintf("profile_id=%v, store token error: %s", settings.Id, err2.Error()))
			return "", model.NewAppError("SmtpSettingsOAuthToken", "store.email.set_token", nil, err2.Error(), http.StatusInternalServerError)
		}
	}
	return newToken.AccessToken, nil
}

func (a *Adapter) ReplyEmail(conn model.EmailConnection, text string) error {
	replyEmail, err := conn.Reply(text)
	if err != nil {
		return err
	}
	if storeErr := a.store.Email().Save(conn.DomainId(), replyEmail); storeErr != nil {
		return model.NewAppError("ReplyEmail", "store.email.save", nil, storeErr.Error(), http.StatusInternalServerError)
	}
	return nil
}
