package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"wbtest/internal/config"
	"wbtest/internal/interfaces"
	"wbtest/internal/model"
)

type OrderValidator struct {
	config *config.Config
}

func NewOrderValidator() interfaces.OrderValidator {
	return &OrderValidator{
		config: config.Load(),
	}
}

func (v *OrderValidator) Validate(order *model.Order) error {
	if order == nil {
		return errors.New("order is nil")
	}

	var validationErrors []string

	// Валидация основных полей
	if err := v.validateOrderUID(order.OrderUID); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if err := v.validateTrackNumber(order.TrackNumber); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if err := v.validateEntry(order.Entry); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Валидация delivery
	if err := v.validateDelivery(&order.Delivery); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("delivery: %s", err.Error()))
	}

	// Валидация payment
	if err := v.validatePayment(&order.Payment); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("payment: %s", err.Error()))
	}

	// Валидация items
	if err := v.validateItems(order.Items); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("items: %s", err.Error()))
	}

	// Валидация дополнительных полей
	if err := v.validateLocale(order.Locale); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if err := v.validateCustomerID(order.CustomerID); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if err := v.validateDeliveryService(order.DeliveryService); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Валидация даты создания
	if err := v.validateDateCreated(order.DateCreated); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Валидация бизнес-логики
	if err := v.validateBusinessLogic(order); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("business logic: %s", err.Error()))
	}

	if len(validationErrors) > 0 {
		return errors.New("validation failed: " + strings.Join(validationErrors, "; "))
	}

	return nil
}

func (v *OrderValidator) validateOrderUID(orderUID string) error {
	if orderUID == "" {
		return errors.New("order_uid is required")
	}
	if len(orderUID) < v.config.Validation.OrderUIDMinLength || len(orderUID) > v.config.Validation.OrderUIDMaxLength {
		return fmt.Errorf("order_uid length must be between %d and %d characters",
			v.config.Validation.OrderUIDMinLength, v.config.Validation.OrderUIDMaxLength)
	}

	// Проверяем формат UID (должен содержать только буквы, цифры и дефисы)
	uidRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !uidRegex.MatchString(orderUID) {
		return errors.New("order_uid contains invalid characters")
	}

	return nil
}

func (v *OrderValidator) validateTrackNumber(trackNumber string) error {
	if trackNumber == "" {
		return errors.New("track_number is required")
	}
	if len(trackNumber) < v.config.Validation.TrackNumberMinLength || len(trackNumber) > v.config.Validation.TrackNumberMaxLength {
		return fmt.Errorf("track_number length must be between %d and %d characters",
			v.config.Validation.TrackNumberMinLength, v.config.Validation.TrackNumberMaxLength)
	}

	// Проверяем формат трек-номера
	trackRegex := regexp.MustCompile(`^[A-Z0-9]+$`)
	if !trackRegex.MatchString(trackNumber) {
		return errors.New("track_number must contain only uppercase letters and numbers")
	}

	return nil
}

func (v *OrderValidator) validateEntry(entry string) error {
	if entry == "" {
		return errors.New("entry is required")
	}
	if len(entry) < 2 || len(entry) > 10 {
		return errors.New("entry length must be between 2 and 10 characters")
	}

	// Проверяем допустимые значения entry
	validEntries := []string{"WBIL", "WBILMT", "WBILM", "WBILT"}
	isValid := false
	for _, validEntry := range validEntries {
		if entry == validEntry {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("entry must be one of: WBIL, WBILMT, WBILM, WBILT")
	}

	return nil
}

func (v *OrderValidator) validateDelivery(delivery *model.Delivery) error {
	if delivery == nil {
		return errors.New("delivery is required")
	}

	if delivery.Name == "" {
		return errors.New("delivery name is required")
	}

	if len(delivery.Name) < 2 || len(delivery.Name) > 100 {
		return errors.New("delivery name length must be between 2 and 100 characters")
	}

	if delivery.Phone == "" {
		return errors.New("delivery phone is required")
	}

	// Улучшенная валидация телефона
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(delivery.Phone) {
		return errors.New("delivery phone format is invalid (must be international format)")
	}

	if delivery.Email == "" {
		return errors.New("delivery email is required")
	}

	// Улучшенная валидация email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(delivery.Email) {
		return errors.New("delivery email format is invalid")
	}

	if delivery.Address == "" {
		return errors.New("delivery address is required")
	}

	if len(delivery.Address) < 5 || len(delivery.Address) > 200 {
		return errors.New("delivery address length must be between 5 and 200 characters")
	}

	if delivery.City == "" {
		return errors.New("delivery city is required")
	}

	if len(delivery.City) < 2 || len(delivery.City) > 50 {
		return errors.New("delivery city length must be between 2 and 50 characters")
	}

	if delivery.Zip != "" {
		zipRegex := regexp.MustCompile(`^\d{5,10}$`)
		if !zipRegex.MatchString(delivery.Zip) {
			return errors.New("delivery zip code format is invalid")
		}
	}

	if delivery.Region != "" {
		if len(delivery.Region) < 2 || len(delivery.Region) > 10 {
			return errors.New("delivery region length must be between 2 and 10 characters")
		}
	}

	return nil
}

func (v *OrderValidator) validatePayment(payment *model.Payment) error {
	if payment == nil {
		return errors.New("payment is required")
	}

	if payment.Transaction == "" {
		return errors.New("payment transaction is required")
	}

	if len(payment.Transaction) < 10 || len(payment.Transaction) > 50 {
		return errors.New("payment transaction length must be between 10 and 50 characters")
	}

	if payment.Currency == "" {
		return errors.New("payment currency is required")
	}

	// Проверяем допустимые валюты
	validCurrencies := []string{"USD", "EUR", "RUB", "GBP", "CNY", "JPY"}
	isValidCurrency := false
	for _, currency := range validCurrencies {
		if payment.Currency == currency {
			isValidCurrency = true
			break
		}
	}
	if !isValidCurrency {
		return errors.New("payment currency must be one of: USD, EUR, RUB, GBP, CNY, JPY")
	}

	if payment.Provider == "" {
		return errors.New("payment provider is required")
	}

	// Проверяем допустимые провайдеры
	validProviders := []string{"wbpay", "stripe", "paypal", "square", "adyen"}
	isValidProvider := false
	for _, provider := range validProviders {
		if payment.Provider == provider {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return errors.New("payment provider must be one of: wbpay, stripe, paypal, square, adyen")
	}

	if payment.Amount <= 0 {
		return errors.New("payment amount must be positive")
	}

	if payment.Amount > v.config.Validation.MaxPaymentAmount {
		return fmt.Errorf("payment amount cannot exceed %d", v.config.Validation.MaxPaymentAmount)
	}

	if payment.PaymentDT <= 0 {
		return errors.New("payment_dt must be positive")
	}

	// Проверяем, что дата платежа не в будущем
	paymentTime := time.Unix(int64(payment.PaymentDT), 0)
	if paymentTime.After(time.Now()) {
		return errors.New("payment_dt cannot be in the future")
	}

	if payment.Bank == "" {
		return errors.New("payment bank is required")
	}

	if len(payment.Bank) < 2 || len(payment.Bank) > 20 {
		return errors.New("payment bank length must be between 2 and 20 characters")
	}

	if payment.DeliveryCost < 0 {
		return errors.New("delivery_cost cannot be negative")
	}

	if payment.DeliveryCost > payment.Amount {
		return errors.New("delivery_cost cannot exceed total amount")
	}

	if payment.GoodsTotal < 0 {
		return errors.New("goods_total cannot be negative")
	}

	// Проверяем, что сумма товаров + доставка = общая сумма
	if payment.GoodsTotal+payment.DeliveryCost != payment.Amount {
		return errors.New("goods_total + delivery_cost must equal total amount")
	}

	return nil
}

func (v *OrderValidator) validateItems(items []model.Item) error {
	if len(items) == 0 {
		return errors.New("at least one item is required")
	}

	if len(items) > v.config.Validation.MaxItemsPerOrder {
		return fmt.Errorf("cannot have more than %d items in order", v.config.Validation.MaxItemsPerOrder)
	}

	totalItemsPrice := 0
	for i, item := range items {
		if err := v.validateItem(&item, i); err != nil {
			return err
		}
		totalItemsPrice += item.TotalPrice
	}

	return nil
}

func (v *OrderValidator) validateItem(item *model.Item, index int) error {
	if item == nil {
		return fmt.Errorf("item[%d] is nil", index)
	}

	if item.ChrtID <= 0 {
		return fmt.Errorf("item[%d] chrt_id must be positive", index)
	}

	if item.ChrtID > 999999999 {
		return fmt.Errorf("item[%d] chrt_id is too large", index)
	}

	if item.Name == "" {
		return fmt.Errorf("item[%d] name is required", index)
	}

	if len(item.Name) < 1 || len(item.Name) > 200 {
		return fmt.Errorf("item[%d] name length must be between 1 and 200 characters", index)
	}

	if item.Price < 0 {
		return fmt.Errorf("item[%d] price cannot be negative", index)
	}

	if item.Price > v.config.Validation.MaxItemPrice {
		return fmt.Errorf("item[%d] price cannot exceed %d", index, v.config.Validation.MaxItemPrice)
	}

	if item.TotalPrice < 0 {
		return fmt.Errorf("item[%d] total_price cannot be negative", index)
	}

	if item.TotalPrice > item.Price {
		return fmt.Errorf("item[%d] total_price cannot exceed price", index)
	}

	if item.NmID <= 0 {
		return fmt.Errorf("item[%d] nm_id must be positive", index)
	}

	if item.NmID > 999999999 {
		return fmt.Errorf("item[%d] nm_id is too large", index)
	}

	if item.Brand == "" {
		return fmt.Errorf("item[%d] brand is required", index)
	}

	if len(item.Brand) < 1 || len(item.Brand) > 50 {
		return fmt.Errorf("item[%d] brand length must be between 1 and 50 characters", index)
	}

	if item.Sale < 0 || item.Sale > 100 {
		return fmt.Errorf("item[%d] sale must be between 0 and 100", index)
	}

	if item.Status < 0 || item.Status > 999 {
		return fmt.Errorf("item[%d] status must be between 0 and 999", index)
	}

	return nil
}

func (v *OrderValidator) validateLocale(locale string) error {
	if locale == "" {
		return errors.New("locale is required")
	}
	if len(locale) != 2 {
		return errors.New("locale must be 2 characters")
	}

	// Проверяем допустимые локали
	validLocales := []string{"en", "ru", "es", "fr", "de", "it", "pt", "ja", "ko", "zh"}
	isValid := false
	for _, validLocale := range validLocales {
		if locale == validLocale {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("locale must be one of: en, ru, es, fr, de, it, pt, ja, ko, zh")
	}

	return nil
}

func (v *OrderValidator) validateCustomerID(customerID string) error {
	if customerID == "" {
		return errors.New("customer_id is required")
	}
	if len(customerID) < 3 || len(customerID) > 20 {
		return errors.New("customer_id length must be between 3 and 20 characters")
	}

	// Проверяем формат customer_id
	customerRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !customerRegex.MatchString(customerID) {
		return errors.New("customer_id contains invalid characters")
	}

	return nil
}

func (v *OrderValidator) validateDeliveryService(deliveryService string) error {
	if deliveryService == "" {
		return errors.New("delivery_service is required")
	}
	if len(deliveryService) < 2 || len(deliveryService) > 20 {
		return errors.New("delivery_service length must be between 2 and 20 characters")
	}

	// Проверяем допустимые службы доставки
	validServices := []string{"meest", "cdek", "dhl", "fedex", "ups", "usps", "ems"}
	isValid := false
	for _, service := range validServices {
		if deliveryService == service {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("delivery_service must be one of: meest, cdek, dhl, fedex, ups, usps, ems")
	}

	return nil
}

func (v *OrderValidator) validateDateCreated(dateCreated time.Time) error {
	if dateCreated.IsZero() {
		return errors.New("date_created is required")
	}

	// Проверяем, что дата не в будущем
	if dateCreated.After(time.Now()) {
		return errors.New("date_created cannot be in the future")
	}

	// Проверяем, что дата не слишком старая (не старше 1 года)
	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	if dateCreated.Before(oneYearAgo) {
		return errors.New("date_created cannot be older than 1 year")
	}

	return nil
}

func (v *OrderValidator) validateBusinessLogic(order *model.Order) error {
	// Проверяем, что сумма всех товаров соответствует payment.goods_total
	totalItemsPrice := 0
	for _, item := range order.Items {
		totalItemsPrice += item.TotalPrice
	}

	if totalItemsPrice != order.Payment.GoodsTotal {
		return fmt.Errorf("sum of items total_price (%d) does not match payment goods_total (%d)",
			totalItemsPrice, order.Payment.GoodsTotal)
	}

	// Проверяем, что transaction в payment соответствует order_uid
	if order.Payment.Transaction != order.OrderUID {
		return errors.New("payment transaction must match order_uid")
	}

	return nil
}
