package validator

import (
	"errors"

	"github.com/go-playground/validator/v10"

	"wbtest/internal/interfaces"
	"wbtest/internal/model"
)

type OrderValidator struct {
	validator *validator.Validate
}

func NewOrderValidator() interfaces.OrderValidator {
	return &OrderValidator{
		validator: validator.New(),
	}
}

func (v *OrderValidator) Validate(order *model.Order) error {
	if order == nil {
		return errors.New("order is nil")
	}

	return v.validator.Struct(order)
}
