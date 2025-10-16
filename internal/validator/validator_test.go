package validator

import (
	"testing"
	"time"

	apperrors "wbtest/internal/errors"
	"wbtest/internal/model"
)

func TestOrderValidator_Validate(t *testing.T) {
	validator := NewOrderValidator()

	tests := []struct {
		name    string
		order   *model.Order
		wantErr bool
	}{
		{
			name: "valid order",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "WBILMTESTTRACK",
				Entry:       "WBIL",
				Delivery: model.Delivery{
					Name:    "Test Testov",
					Phone:   "+9720000000",
					Zip:     "2639809",
					City:    "Kiryat Mozkin",
					Address: "Ploshad Mira 15",
					Region:  "Kraiot",
					Email:   "test@gmail.com",
				},
				Payment: model.Payment{
					Transaction:  "b563feb7b2b84b6test",
					RequestID:    "",
					Currency:     "USD",
					Provider:     "wbpay",
					Amount:       1817,
					PaymentDT:    1637907727,
					Bank:         "alpha",
					DeliveryCost: 1500,
					GoodsTotal:   317,
					CustomFee:    0,
				},
				Items: []model.Item{
					{
						ChrtID:      9934930,
						TrackNumber: "WBILMTESTTRACK",
						Price:       453,
						Rid:         "ab4219087a764ae0btest",
						Name:        "Mascaras",
						Sale:        30,
						Size:        "0",
						TotalPrice:  317,
						NmID:        2389212,
						Brand:       "Vivienne Sabo",
						Status:      202,
					},
				},
				Locale:            "en",
				InternalSignature: "",
				CustomerID:        "test",
				DeliveryService:   "meest",
				ShardKey:          "9",
				SmID:              99,
				DateCreated:       time.Now(),
				OofShard:          "1",
			},
			wantErr: false,
		},
		{
			name:    "nil order",
			order:   nil,
			wantErr: true,
		},
		{
			name: "invalid order - empty order_uid",
			order: &model.Order{
				OrderUID: "",
			},
			wantErr: true,
		},
		{
			name: "invalid order - short order_uid",
			order: &model.Order{
				OrderUID: "short",
			},
			wantErr: true,
		},
		{
			name: "invalid order - long order_uid",
			order: &model.Order{
				OrderUID: "this-is-a-very-long-order-uid-that-exceeds-the-maximum-allowed-length",
			},
			wantErr: true,
		},
		{
			name: "invalid order - empty track_number",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "",
			},
			wantErr: true,
		},
		{
			name: "invalid order - short track_number",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "WB",
			},
			wantErr: true,
		},
		{
			name: "invalid order - empty entry",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "WBILMTESTTRACK",
				Entry:       "",
			},
			wantErr: true,
		},
		{
			name: "invalid order - invalid email",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "WBILMTESTTRACK",
				Entry:       "WBIL",
				Delivery: model.Delivery{
					Name:    "Test Testov",
					Phone:   "+9720000000",
					Zip:     "2639809",
					City:    "Kiryat Mozkin",
					Address: "Ploshad Mira 15",
					Region:  "Kraiot",
					Email:   "invalid-email",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid order - empty items",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "WBILMTESTTRACK",
				Entry:       "WBIL",
				Delivery: model.Delivery{
					Name:    "Test Testov",
					Phone:   "+9720000000",
					Zip:     "2639809",
					City:    "Kiryat Mozkin",
					Address: "Ploshad Mira 15",
					Region:  "Kraiot",
					Email:   "test@gmail.com",
				},
				Items: []model.Item{},
			},
			wantErr: true,
		},
		{
			name: "invalid order - invalid currency",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "WBILMTESTTRACK",
				Entry:       "WBIL",
				Delivery: model.Delivery{
					Name:    "Test Testov",
					Phone:   "+9720000000",
					Zip:     "2639809",
					City:    "Kiryat Mozkin",
					Address: "Ploshad Mira 15",
					Region:  "Kraiot",
					Email:   "test@gmail.com",
				},
				Payment: model.Payment{
					Transaction:  "b563feb7b2b84b6test",
					RequestID:    "",
					Currency:     "US", // Invalid currency (should be 3 chars)
					Provider:     "wbpay",
					Amount:       1817,
					PaymentDT:    1637907727,
					Bank:         "alpha",
					DeliveryCost: 1500,
					GoodsTotal:   317,
					CustomFee:    0,
				},
				Items: []model.Item{
					{
						ChrtID:      9934930,
						TrackNumber: "WBILMTESTTRACK",
						Price:       453,
						Rid:         "ab4219087a764ae0btest",
						Name:        "Mascaras",
						Sale:        30,
						Size:        "0",
						TotalPrice:  317,
						NmID:        2389212,
						Brand:       "Vivienne Sabo",
						Status:      202,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.order)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Проверяем тип ошибки для nil order
			if tt.name == "nil order" && err != nil {
				if appErr, ok := err.(*apperrors.AppError); !ok {
					t.Errorf("Expected AppError for nil order, got %T", err)
				} else if appErr.Type != apperrors.ErrorTypeValidation {
					t.Errorf("Expected ErrorTypeValidation, got %s", appErr.Type)
				}
			}
		})
	}
}
