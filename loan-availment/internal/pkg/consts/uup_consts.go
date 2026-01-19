package consts

type BrandType string
type CustomerType string

// Brand constants
const (
	BrandGHPPrepaid BrandType = "GHP-PREPAID"
	BrandTM         BrandType = "TM"
	BrandGHP        BrandType = "GHP"
	BrandPrepaidGHP BrandType = "Prepaid(GHP)"
	BrandGP                   = "GP"
)

// Customer Type constants
const (
	CustomerConsumer CustomerType = "Consumer"
	CustomerC        CustomerType = "C"
)

// If you want a collection of these constants, you can use a slice
var BrandTypes = []BrandType{
	BrandGHPPrepaid,
	BrandTM,
	BrandGHP,
	BrandPrepaidGHP,
}

var CustomerTypes = []CustomerType{
	CustomerConsumer,
	CustomerC,
}
