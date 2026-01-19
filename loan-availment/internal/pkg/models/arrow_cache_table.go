package models

type ArrowCacheTable struct {
	CollectionRecordsPublished int64 `json:"collection_records_published"`
	AvailmentRecordsPublished  int64 `json:"availment_records_published"`
	SuccessfulCollections      int64 `json:"successful_collections"`
	SuccessfulAvailments       int64 `json:"successful_availments"`
}
