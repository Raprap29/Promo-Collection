package models

type SubscriberDetail struct {
	Entries []Entry `xml:"Entries>Entry"`
}
