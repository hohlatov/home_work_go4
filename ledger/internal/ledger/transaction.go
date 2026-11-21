package ledger

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Transaction — финансовая транзакция.
type Transaction struct {
	ID          int       // уникальный идентификатор (автоинкремент)
	Amount      float64   // сумма
	Category    string    // категория траты
	Description string    // описание
	Date        time.Time // дата транзакции
}

// Validate проверяет поля транзакции и возвращает все найденные проблемы
// одной ошибкой *ValidationError. Если всё корректно, возвращает nil.
func (t Transaction) Validate() error {
	var problems []error

	if !isPositive(t.Amount) {
		problems = append(problems, fmt.Errorf("%w (получено %v)", ErrInvalidAmount, t.Amount))
	}
	if strings.TrimSpace(t.Category) == "" {
		problems = append(problems, ErrEmptyCategory)
	}
	if t.Date.IsZero() {
		problems = append(problems, ErrMissingDate)
	}

	return newValidationError("транзакция", problems)
}

// isPositive отбрасывает 0, отрицательные числа, NaN и бесконечность.
func isPositive(x float64) bool {
	return x > 0 && !math.IsInf(x, 0) // для NaN сравнение x > 0 ложно
}
