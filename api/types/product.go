package types

// Product is ...
type ProductMetadata struct {
	ID        string  `json:"id" bson:"id"`
	Thumbnail *string `json:"thumbnail" bson:"thumbnail"`
	Nutrition *string `json:"nutritional_facts" bson:"nutritional_facts"`
}

type ProductDataSearch struct {
	Name      string  `json:"name"`
	ID        string  `json:"id"`
	Thumbnail *string `json:"thumbnail"`
	Nutrition *string `json:"nutritional_facts"`
}

type ProductData struct {
	Name      string         `json:"name"`
	ID        string         `json:"id"`
	Thumbnail *string        `json:"thumbnail"`
	Nutrition *string        `json:"nutritional_facts"`
	Amounts   map[string]int `json:"amounts"`
}

type LocationProductData struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Thumbnail *string `json:"thumbnail"`
	Nutrition *string `json:"nutritional_facts"`
	Amount    int     `json:"amount"`
}

type LocationProductDataSearch struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Thumbnail *string `json:"thumbnail"`
	Amount    int     `json:"amount"`
}
