package domain

type Category string

const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryOther         Category = "other"
)

// IsValid проверяет, существует ли такая категория.
func (c Category) IsValid() bool {
	switch c {
	case CategoryFood,
		CategoryTransport,
		CategoryHousing,
		CategoryEntertainment,
		CategoryHealth,
		CategoryShopping,
		CategoryOther:
		return true
	default:
		return false
	}
}
