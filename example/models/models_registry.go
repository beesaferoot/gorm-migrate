package models

var ModelTypeRegistry = map[string]any{
	"Apartment":             Apartment{},
	"ApartmentBookingPrice": ApartmentBookingPrice{},
	"ApartmentContract":     ApartmentContract{},
	"ApartmentHighlight":    ApartmentHighlight{},
	"Estate":                Estate{},
	"Tenant":                Tenant{},
	"User":                  User{},
	"CustomSchema":          CustomSchema{},
}
