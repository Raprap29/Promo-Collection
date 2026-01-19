package consts

const (
	ObjectIDRegexStr     = `^[0-9a-fA-F]{24}$`
	DatetimeRegexStr     = `^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}Z$`
	ValidPrefixForMSISDN = `^(9|09|639|63)\d{9}$`
	ValidChannelCode     = `^[a-zA-Z0-9-_]+$`
	ValidKeywordCode     = `^[a-zA-Z0-9 ]+$`
)
