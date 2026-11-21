package ledger

import (
	"errors"
	"strings"
)

// Validatable — контракт на проверку валидности данных структуры.
// Validate только возвращает ошибку и ничего не логирует:
// решение, что делать с ошибкой, принимает вызывающий код.
type Validatable interface {
	Validate() error
}

// Проверка на этапе компиляции: модели реализуют Validatable.
var (
	_ Validatable = Transaction{}
	_ Validatable = Budget{}
)

// Ошибки валидации. Каждая конкретная проблема оборачивает одну из них,
// поэтому их можно проверять через errors.Is.
var (
	ErrInvalidAmount  = errors.New("сумма должна быть больше 0")
	ErrEmptyCategory  = errors.New("категория не может быть пустой")
	ErrMissingDate    = errors.New("дата не указана")
	ErrInvalidLimit   = errors.New("лимит должен быть больше 0")
	ErrNilValidatable = errors.New("объект для проверки не передан")
)

// CheckValid проверяет любой объект, реализующий Validatable.
// Функции не важно, транзакция это или бюджет: она работает через интерфейс.
func CheckValid(v Validatable) error {
	if v == nil {
		return ErrNilValidatable
	}
	return v.Validate()
}

// ValidationError собирает все проблемы объекта, чтобы пользователь
// увидел их сразу, а не исправлял по одной.
type ValidationError struct {
	Entity   string  // что проверялось, например «транзакция»
	Problems []error // найденные проблемы
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		msgs[i] = p.Error()
	}
	return "некорректная " + e.Entity + ": " + strings.Join(msgs, "; ")
}

// Unwrap позволяет errors.Is/errors.As находить каждую из проблем.
func (e *ValidationError) Unwrap() []error {
	return e.Problems
}

// newValidationError возвращает nil, если проблем нет. Именно nil интерфейса error,
// а не (*ValidationError)(nil): иначе проверка err != nil всегда была бы истинной.
func newValidationError(entity string, problems []error) error {
	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Entity: entity, Problems: problems}
}
