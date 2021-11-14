package biz

import (
	"errors"
	"example/go-web-service/api"
	"example/go-web-service/internal/data"
)

type UserBiz struct {
	api.UnimplementedUserServiceServer
	repo *data.UserRepo
}

func NewUserBiz(repo *data.UserRepo) *UserBiz {
	return &UserBiz{
		repo: repo,
	}
}

func (ub *UserBiz) GetUserById(uid uint64) (*UserDO, error) {
	if uid == 0 {
		// 考虑使用错误码 - 内部业务错误码
		return nil, errors.New("invalid user id")
	}
	u, err := ub.repo.GetUser(uid)
	if err != nil {
		// 理论上来说，repo 会把 error 组装好，附加上各种必要的debug 信息，这里可以直接返回
		// 如果 repo 里面并没有处理，依旧是保留着原生的 DB 错误数据，这边要考虑转换具体业务错误
		// 比如说 NoRows 这种错误，要考虑转换为 user not found 或者 invalid user id
		return nil, err
	}
	return &UserDO{
		NickName: u.NickName,
	}, nil
}

type UserDO struct {
	NickName string
}

type MyService interface {

}

type service struct {

}

type Option func(db *service) MyService

func NewDB(opts...Option) MyService {
	return &service{}
}