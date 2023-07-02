package models

import (
	"errors"
)

var (
  ErrNoRecord = errors.New("models: no matching record found")
  ErrInvalidCredentials = errors.New("models: invalid credentials")
  ErrDuplicateElemail = errors.New("models: duplicate email")
) 
