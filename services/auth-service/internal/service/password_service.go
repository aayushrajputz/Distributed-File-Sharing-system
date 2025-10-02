package service

import (
	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct {
	cost int
}

func NewPasswordService() *PasswordService {
	return &PasswordService{
		cost: bcrypt.DefaultCost,
	}
}

func (s *PasswordService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	return string(bytes), err
}

func (s *PasswordService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
