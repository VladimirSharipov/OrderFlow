package validator

import (
	"testing"
	"time"

	"wbtest/internal/model"
)

func TestOrderValidator_Validate_ValidOrder(t *testing.T) {
	validator := NewOrderValidator()

	order := createValidOrder()

	err := validator.Validate(order)
	if err != nil {
		t.Errorf("Expected valid order to pass validation, got error: %v", err)
	}
}

func TestOrderValidator_Validate_NilOrder(t *testing.T) {
	validator := NewOrderValidator()

	err := validator.Validate(nil)
	if err == nil {
		t.Error("Expected nil order to fail validation")
	}
	if err.Error() != "order is nil" {
		t.Errorf("Expected 'order is nil' error, got: %v", err)
	}
}

func TestOrderValidator_ValidateOrderUID(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name     string
		orderUID string
		wantErr  bool
	}{
		{"empty", "", true},
		{"too short", "123", true},
		{"too long", "123456789012345678901234567890123456789012345678901234567890", true},
		{"invalid chars", "order@123", true},
		{"valid", "order_123_test", false},
		{"valid with numbers", "order123test", false},
		{"valid with hyphens", "order-123-test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateOrderUID(tt.orderUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOrderUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateTrackNumber(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name        string
		trackNumber string
		wantErr     bool
	}{
		{"empty", "", true},
		{"too short", "1234", true},
		{"too long", "TRACK12345678901234567890", true},
		{"invalid chars", "track@123", true},
		{"lowercase", "track123", true},
		{"valid", "TRACK123", false},
		{"valid with numbers", "TRACK123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateTrackNumber(tt.trackNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTrackNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateEntry(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "W", true},
		{"too long", "WBILMTEST", true},
		{"invalid", "INVALID", true},
		{"valid WBIL", "WBIL", false},
		{"valid WBILMT", "WBILMT", false},
		{"valid WBILM", "WBILM", false},
		{"valid WBILT", "WBILT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateDelivery(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name     string
		delivery *model.Delivery
		wantErr  bool
	}{
		{"nil", nil, true},
		{"empty name", &model.Delivery{Name: "", Phone: "+1234567890", Email: "test@example.com", Address: "123 Main St", City: "Test City"}, true},
		{"empty phone", &model.Delivery{Name: "John Doe", Phone: "", Email: "test@example.com", Address: "123 Main St", City: "Test City"}, true},
		{"invalid phone", &model.Delivery{Name: "John Doe", Phone: "invalid", Email: "test@example.com", Address: "123 Main St", City: "Test City"}, true},
		{"empty email", &model.Delivery{Name: "John Doe", Phone: "+1234567890", Email: "", Address: "123 Main St", City: "Test City"}, true},
		{"invalid email", &model.Delivery{Name: "John Doe", Phone: "+1234567890", Email: "invalid-email", Address: "123 Main St", City: "Test City"}, true},
		{"empty address", &model.Delivery{Name: "John Doe", Phone: "+1234567890", Email: "test@example.com", Address: "", City: "Test City"}, true},
		{"empty city", &model.Delivery{Name: "John Doe", Phone: "+1234567890", Email: "test@example.com", Address: "123 Main St", City: ""}, true},
		{"invalid zip", &model.Delivery{Name: "John Doe", Phone: "+1234567890", Email: "test@example.com", Address: "123 Main St", City: "Test City", Zip: "invalid"}, true},
		{"valid", &model.Delivery{Name: "John Doe", Phone: "+1234567890", Email: "test@example.com", Address: "123 Main St", City: "Test City", Zip: "12345"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDelivery(tt.delivery)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDelivery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidatePayment(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name    string
		payment *model.Payment
		wantErr bool
	}{
		{"nil", nil, true},
		{"empty transaction", &model.Payment{Transaction: "", Currency: "USD", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test"}, true},
		{"invalid currency", &model.Payment{Transaction: "test123", Currency: "INVALID", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test"}, true},
		{"invalid provider", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "invalid", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test"}, true},
		{"negative amount", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: -100, PaymentDT: int(time.Now().Unix()), Bank: "test"}, true},
		{"zero amount", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: 0, PaymentDT: int(time.Now().Unix()), Bank: "test"}, true},
		{"too large amount", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: 2000000, PaymentDT: int(time.Now().Unix()), Bank: "test"}, true},
		{"future payment date", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().AddDate(0, 0, 1).Unix()), Bank: "test"}, true},
		{"negative delivery cost", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test", DeliveryCost: -50}, true},
		{"delivery cost exceeds amount", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test", DeliveryCost: 150}, true},
		{"invalid total calculation", &model.Payment{Transaction: "test123", Currency: "USD", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test", DeliveryCost: 20, GoodsTotal: 90}, true},
		{"valid", &model.Payment{Transaction: "test123456789", Currency: "USD", Provider: "stripe", Amount: 100, PaymentDT: int(time.Now().Unix()), Bank: "test", DeliveryCost: 20, GoodsTotal: 80}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePayment(tt.payment)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePayment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateItems(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name    string
		items   []model.Item
		wantErr bool
	}{
		{"empty", []model.Item{}, true},
		{"too many items", make([]model.Item, 101), true},
		{"valid", []model.Item{createValidItem()}, false},
		{"valid", []model.Item{createValidItem()}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateItems(tt.items)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateItems() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateItem(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name    string
		item    *model.Item
		index   int
		wantErr bool
	}{
		{"nil", nil, 0, true},
		{"invalid chrt_id", &model.Item{ChrtID: 0, Name: "Test", Price: 100, TotalPrice: 100, NmID: 123456, Brand: "Test"}, 0, true},
		{"empty name", &model.Item{ChrtID: 123456, Name: "", Price: 100, TotalPrice: 100, NmID: 123456, Brand: "Test"}, 0, true},
		{"negative price", &model.Item{ChrtID: 123456, Name: "Test", Price: -100, TotalPrice: 100, NmID: 123456, Brand: "Test"}, 0, true},
		{"negative total price", &model.Item{ChrtID: 123456, Name: "Test", Price: 100, TotalPrice: -100, NmID: 123456, Brand: "Test"}, 0, true},
		{"total price exceeds price", &model.Item{ChrtID: 123456, Name: "Test", Price: 100, TotalPrice: 150, NmID: 123456, Brand: "Test"}, 0, true},
		{"invalid nm_id", &model.Item{ChrtID: 123456, Name: "Test", Price: 100, TotalPrice: 100, NmID: 0, Brand: "Test"}, 0, true},
		{"empty brand", &model.Item{ChrtID: 123456, Name: "Test", Price: 100, TotalPrice: 100, NmID: 123456, Brand: ""}, 0, true},
		{"invalid sale", &model.Item{ChrtID: 123456, Name: "Test", Price: 100, TotalPrice: 100, NmID: 123456, Brand: "Test", Sale: 150}, 0, true},
		{"invalid status", &model.Item{ChrtID: 123456, Name: "Test", Price: 100, TotalPrice: 100, NmID: 123456, Brand: "Test", Status: 1000}, 0, true},
		{"valid", func() *model.Item { item := createValidItem(); return &item }(), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateItem(tt.item, tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateLocale(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name    string
		locale  string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "e", true},
		{"too long", "eng", true},
		{"invalid", "xx", true},
		{"valid en", "en", false},
		{"valid ru", "ru", false},
		{"valid es", "es", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateLocale(tt.locale)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLocale() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateCustomerID(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name       string
		customerID string
		wantErr    bool
	}{
		{"empty", "", true},
		{"too short", "ab", true},
		{"too long", "customer12345678901234567890", true},
		{"invalid chars", "customer@123", true},
		{"valid", "customer123", false},
		{"valid with underscore", "customer_123", false},
		{"valid with hyphen", "customer-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateCustomerID(tt.customerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCustomerID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateDeliveryService(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name            string
		deliveryService string
		wantErr         bool
	}{
		{"empty", "", true},
		{"too short", "a", true},
		{"too long", "deliveryservice12345678901234567890", true},
		{"invalid", "invalid", true},
		{"valid meest", "meest", false},
		{"valid cdek", "cdek", false},
		{"valid dhl", "dhl", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDeliveryService(tt.deliveryService)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDeliveryService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateDateCreated(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name        string
		dateCreated time.Time
		wantErr     bool
	}{
		{"zero time", time.Time{}, true},
		{"future date", time.Now().AddDate(0, 0, 1), true},
		{"too old", time.Now().AddDate(-2, 0, 0), true},
		{"valid", time.Now().AddDate(0, 0, -1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDateCreated(tt.dateCreated)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDateCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderValidator_ValidateBusinessLogic(t *testing.T) {
	validator := &OrderValidator{}

	tests := []struct {
		name    string
		order   *model.Order
		wantErr bool
	}{
		{"mismatched transaction", &model.Order{
			OrderUID: "order123",
			Payment:  model.Payment{Transaction: "different123", GoodsTotal: 80},
			Items:    []model.Item{{TotalPrice: 80}},
		}, true},
		{"mismatched goods total", &model.Order{
			OrderUID: "order123",
			Payment:  model.Payment{Transaction: "order123", GoodsTotal: 100},
			Items:    []model.Item{{TotalPrice: 80}},
		}, true},
		{"valid", &model.Order{
			OrderUID: "order123",
			Payment:  model.Payment{Transaction: "order123", GoodsTotal: 80},
			Items:    []model.Item{{TotalPrice: 80}},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateBusinessLogic(tt.order)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBusinessLogic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Вспомогательные функции для создания тестовых данных
func createValidOrder() *model.Order {
	return &model.Order{
		OrderUID:        "order_123_test",
		TrackNumber:     "TRACK123",
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      "customer123",
		DeliveryService: "dhl",
		DateCreated:     time.Now().AddDate(0, 0, -1),
		Delivery: model.Delivery{
			Name:    "John Doe",
			Phone:   "+1234567890",
			Email:   "test@example.com",
			Address: "123 Main St",
			City:    "Test City",
			Zip:     "12345",
		},
		Payment: model.Payment{
			Transaction:  "order_123_test",
			Currency:     "USD",
			Provider:     "stripe",
			Amount:       100,
			PaymentDT:    int(time.Now().Unix()),
			Bank:         "test",
			DeliveryCost: 20,
			GoodsTotal:   80,
		},
		Items: []model.Item{createValidItem()},
	}
}

func createValidItem() model.Item {
	return model.Item{
		ChrtID:     123456,
		Name:       "Test Item",
		Price:      100,
		TotalPrice: 80,
		NmID:       654321,
		Brand:      "Test Brand",
		Sale:       20,
		Status:     202,
	}
}
