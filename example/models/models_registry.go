package models

var ModelTypeRegistry = map[string]interface{}{
	"Apartment":             Apartment{},
	"ApartmentBookingPrice": ApartmentBookingPrice{},
	"ApartmentContract":     ApartmentContract{},
	"ApartmentHighlight":    ApartmentHighlight{},
	"Estate":                Estate{},
	"Tenant":                Tenant{},
	"User":                  User{},
}
