package domain

import (
	"time"

	"github.com/google/uuid"
)

// Тариф
type Plan struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name                string    `gorm:"unique"` // "Базовый", "Стандарт"
	Price               int       // Цена в рублях
	Description         string
	CourseLimit         int  // 1, 5, -1 (безлимит)
	SnowflakePrice      int  // <<-- ДОБАВЬТЕ ЭТУ СТРОКУ (Цена в снежинках)
	DeviceLimit         int  // 2, 4
	IsTgAccess          bool // Доступ к закрытому ТГ
	DefaultDurationDays int  // 30

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Промокод
type PromoCode struct {
	Code string `gorm:"primaryKey"` // "START3"

	// К какому тарифу привязан код
	PlanID *uuid.UUID `gorm:"type:uuid"`
	Plan   Plan       `gorm:"foreignKey:PlanID"`

	Type string
	// Если ONE_COURSE, тут кол-во слотов. Если SUBSCRIPTION - PlanID
	ValueInt int

	// Условия
	OverrideDuration int        // Если > 0, заменяет стандартную длительность (например, 3 дня)
	MaxUses          int        // Сколько раз можно использовать всего
	UsedCount        int        // Текущее использование
	ExpiresAt        *time.Time // Дата сгорания кода (может быть null)
}

type PromoActivation struct {
	UserID    string `gorm:"primaryKey;index"` // ID пользователя
	Code      string `gorm:"primaryKey;index"` // Сам код (например "START3")
	CreatedAt time.Time
}
