package models

import "reflect"

var ModelTypeRegistry = map[string]reflect.Type{
	"Apartment": reflect.TypeOf(Apartment{}),
	"ApartmentBookingPrice": reflect.TypeOf(ApartmentBookingPrice{}),
	"ApartmentContract": reflect.TypeOf(ApartmentContract{}),
	"ApartmentHighlight": reflect.TypeOf(ApartmentHighlight{}),
	"Estate": reflect.TypeOf(Estate{}),
	"Tenant": reflect.TypeOf(Tenant{}),
	"User": reflect.TypeOf(User{}),
}
