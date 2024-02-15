package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"

	"github.com/webitel/flow_manager/model"
)

var smtpOauth2Group singleflight.Group

func (fm *FlowManager) SmtpSettingsOAuthToken(settings *model.SmtSettings) (string, *model.AppError) {
	token, err, _ := smtpOauth2Group.Do(fmt.Sprintf("%v", settings.Id), func() (interface{}, error) {
		token, err := fm.smtpSettingsOAuthToken(settings)
		if err != nil {
			return token, err
		}

		return token, nil
	})

	if err != nil {
		switch err.(type) {
		case *model.AppError:
			return "", err.(*model.AppError)
		default:
			return "", model.NewAppError("App", "app.smtp.oauth", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	return token.(string), nil
}

func (fm *FlowManager) smtpSettingsOAuthToken(settings *model.SmtSettings) (string, *model.AppError) {
	if settings.Params == nil || settings.Params.OAuth2 == nil {
		// TODO
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
		if err2 := fm.Store.Email().SetToken(settings.Id, newToken); err2 != nil {
			wlog.Error(fmt.Sprintf("profile_id=%v, store token error: %s", settings.Id, err2.Error()))
			return "", err2
		}
	}

	return newToken.AccessToken, nil
}
