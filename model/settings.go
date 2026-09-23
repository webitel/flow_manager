package model

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	SysAutoLinkCallToContact = "autolink_call_to_contact"
	SysAutoLinkMailToContact = "autolink_mail_to_contact"
)

const SystemSettingsObjectName = "system_settings"

type SysValue struct {
	BoolValue   bool
	StringValue string
}

type SystemSettingEvent struct {
	Name     string
	DomainID int64
}

func NewSystemSettingEventFromRoutingKey(rk string) (*SystemSettingEvent, *AppError) {
	parts := strings.Split(rk, ".")
	if len(parts) < 4 {
		return nil, NewRequestError(
			"model.settings.new_system_setting_event.invalid_rk_len",
			fmt.Sprintf("received routing key %q with len less than 4", rk),
		)
	}

	if parts[0] != SystemSettingsObjectName || parts[1] == "" {
		return nil, NewRequestError(
			"model.settings.new_system_setting_event.invalid_rk",
			fmt.Sprintf("received unexpected routing key %q", rk),
		)
	}

	domainID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || domainID <= 0 {
		return nil, NewRequestError(
			"model.settings.new_system_setting_event.invalid_domain_id",
			fmt.Sprintf("received invalid domain id in routing key %q", rk),
		)
	}

	return &SystemSettingEvent{Name: parts[1], DomainID: domainID}, nil
}

func SystemSettingCacheKey(domainID int64, name string) string {
	return fmt.Sprintf("%d-%s", domainID, name)
}
