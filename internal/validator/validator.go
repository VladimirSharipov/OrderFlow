package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	apperrors "wbtest/internal/errors"
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
		return apperrors.New(apperrors.ErrorTypeValidation, "order is nil")
	}

	err := v.validator.Struct(order)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errorMessages []string
			for _, validationErr := range validationErrors {
				errorMessages = append(errorMessages, fmt.Sprintf("field '%s' failed validation: %s", validationErr.Field(), validationErr.Tag()))
			}
			return apperrors.NewWithCode(
				apperrors.ErrorTypeValidation,
				"validation failed: "+strings.Join(errorMessages, "; "),
				"VALIDATION_FAILED",
			)
		}
		return apperrors.Wrap(err, apperrors.ErrorTypeValidation, "validation error")
	}

	return nil
}
