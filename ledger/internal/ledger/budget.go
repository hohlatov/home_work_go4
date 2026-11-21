package ledger

import (
	"fmt"
	"strings"
)

// Budget — лимит расходов на категорию.
type Budget struct {
	Category string  `json:"category"` // категория, на которую выделен бюджет
	Limit    float64 `json:"limit"`    // лимит бюджета
}

// Validate проверяет поля бюджета и возвращает все найденные проблемы
// одной ошибкой *ValidationError. Если всё корректно, возвращает nil.
func (b Budget) Validate() error {
	var problems []error

	if strings.TrimSpace(b.Category) == "" {
		problems = append(problems, ErrEmptyCategory)
	}
	if !isPositive(b.Limit) {
		problems = append(problems, fmt.Errorf("%w (получено %v)", ErrInvalidLimit, b.Limit))
	}

	return newValidationError("запись бюджета", problems)
}
