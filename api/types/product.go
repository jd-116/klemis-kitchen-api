package types

// ProductMetadata contains the data stored in MongoDB
// that includes the additional product data, such as thumbnail and nutritional facts
type ProductMetadata struct {
	ID        string  `json:"id" bson:"id"`
	Thumbnail *string `json:"thumbnail" bson:"thumbnail"`
	Nutrition *string `json:"nutritional_facts" bson:"nutritional_facts"`
}

// ProductDataSearch is the result of a full product with the amounts map omitted,
// used in large collections of products
type ProductDataSearch struct {
	Name      string  `json:"name"`
	ID        string  `json:"id"`
	Thumbnail *string `json:"thumbnail"`
	Nutrition *string `json:"nutritional_facts"`
}

// ProductData is the result of a full product,
// used when retrieving a single product
type ProductData struct {
	Name      string         `json:"name"`
	ID        string         `json:"id"`
	Thumbnail *string        `json:"thumbnail"`
	Nutrition *string        `json:"nutritional_facts"`
	Amounts   map[string]int `json:"amounts"`
}

// LocationProductDataSearch is the result of a full product with the amount number omitted,
// used in large collections of products
type LocationProductDataSearch struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Thumbnail *string `json:"thumbnail"`
	Amount    int     `json:"amount"`
}

// LocationProductData is the result of a full product,
// used when retrieving a single product at a location
type LocationProductData struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Thumbnail *string `json:"thumbnail"`
	Nutrition *string `json:"nutritional_facts"`
	Amount    int     `json:"amount"`
}
