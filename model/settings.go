package model

const (
	SysAutoLinkCallToContact = "autolink_call_to_contact"
	SysAutoLinkMailToContact = "autolink_mail_to_contact"
	SysRecordAllCalls        = "record_all_calls"
)

type SysValue struct {
	BoolValue   bool
	StringValue string
}
