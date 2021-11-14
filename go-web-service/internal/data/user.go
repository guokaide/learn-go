package data

import (
	"encoding/json"
	"github.com/gotomicro/ego-component/egorm"
	"github.com/gotomicro/ego-component/eredis"
)

type UserRepo struct {
	db    *egorm.Component
	cache *eredis.Component
}

type DB struct {
	dsn      string
	username string
	password string
	cfg      DBConfig
}

type DBConfig struct {
	dsn      string
	username string
	password string
}

func NewUserRepo(db *egorm.Component, cache *eredis.Component) *UserRepo {
	return &UserRepo{
		db:    db,
		cache: cache,
	}
}

func (u *UserRepo) GetUser(id uint64) (*UserPO, error) {
	return &UserPO{
		NickName: "Tom",
	}, nil
}

// UserPO Persistent Object - data
type UserPO struct {
	NickName string
}

// UserDO Domain Object - biz
type UserDO struct {
	NickName string
}

func convert() {
	bytes, _ := json.Marshal(&UserPO{})
	do := &UserDO{}
	json.Unmarshal(bytes, do)
}
