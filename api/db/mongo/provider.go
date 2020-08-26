package mongo

import (
	"log"

	"github.com/jd-116/klemis-kitchen-api/util"
)

type Provider struct{}

func NewProvider() (*Provider, error) {
	dbPort, err := util.GetIntEnv("MongoDB port", "MONGO_DB_PORT")
	if err != nil {
		return nil, err
	}

	dbHost, err := util.GetEnv("MongoDB host", "MONGO_DB_HOST")
	if err != nil {
		log.Println(err)
		return nil, err
	}
}
