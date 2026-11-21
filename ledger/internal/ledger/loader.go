package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// LoadBudgets читает из r JSON-массив бюджетов и добавляет их через SetBudget.
//
// Формат:
//
//	[{"category": "Еда", "limit": 5000}, {"category": "Транспорт", "limit": 3000}]
//
// Сначала каждый элемент проверяется через Validate. Если хотя бы один
// некорректен, бюджеты не применяются, чтобы хранилище не осталось
// заполненным наполовину.
func LoadBudgets(r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields() // опечатка вроде "limt" должна давать ошибку, а не нулевой лимит

	var list []Budget
	if err := dec.Decode(&list); err != nil {
		return describeJSONError(err)
	}
	// После массива не должно быть других данных.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("загрузка бюджетов: после JSON-массива есть лишние данные")
	}
	if list == nil {
		return errors.New("загрузка бюджетов: ожидался JSON-массив, получено null")
	}

	for i, b := range list {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("загрузка бюджетов: элемент #%d: %w", i+1, err)
		}
	}
	for i, b := range list {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("загрузка бюджетов: элемент #%d: %w", i+1, err)
		}
	}
	return nil
}

// describeJSONError переводит ошибки encoding/json в понятные сообщения.
func describeJSONError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.Is(err, io.EOF):
		return errors.New("загрузка бюджетов: пустые входные данные")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("загрузка бюджетов: JSON обрывается раньше конца")
	case errors.As(err, &syntaxErr):
		return fmt.Errorf("загрузка бюджетов: синтаксическая ошибка JSON на позиции %d: %w", syntaxErr.Offset, err)
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return fmt.Errorf("загрузка бюджетов: ожидался JSON-массив объектов, получено %s", typeErr.Value)
		}
		// В новых версиях Go путь содержит индекс элемента массива: "0.limit".
		where, field := "", typeErr.Field
		if idx, rest, ok := strings.Cut(field, "."); ok {
			if n, err := strconv.Atoi(idx); err == nil {
				where, field = fmt.Sprintf("элемент #%d: ", n+1), rest
			}
		}
		return fmt.Errorf("загрузка бюджетов: %sполе %q должно иметь тип %s, получено %s",
			where, field, typeErr.Type, typeErr.Value)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("загрузка бюджетов: неизвестное поле %s (допустимы: category, limit)", field)
	default:
		return fmt.Errorf("загрузка бюджетов: %w", err)
	}
}
