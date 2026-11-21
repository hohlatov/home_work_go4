package ledger

import (
	"errors"
	"math"
	"testing"
	"time"
)

var someDate = time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

func TestTransactionValidate(t *testing.T) {
	cases := []struct {
		name string
		tx   Transaction
		want []error // nil — транзакция корректна
	}{
		{"корректная", Transaction{Amount: 100, Category: "Еда", Date: someDate}, nil},
		{"копейки", Transaction{Amount: 0.01, Category: "Еда", Date: someDate}, nil},
		{"нулевая сумма", Transaction{Amount: 0, Category: "Еда", Date: someDate}, []error{ErrInvalidAmount}},
		{"отрицательная сумма", Transaction{Amount: -10, Category: "Еда", Date: someDate}, []error{ErrInvalidAmount}},
		{"NaN", Transaction{Amount: math.NaN(), Category: "Еда", Date: someDate}, []error{ErrInvalidAmount}},
		{"бесконечность", Transaction{Amount: math.Inf(1), Category: "Еда", Date: someDate}, []error{ErrInvalidAmount}},
		{"пустая категория", Transaction{Amount: 100, Category: "", Date: someDate}, []error{ErrEmptyCategory}},
		{"категория из пробелов", Transaction{Amount: 100, Category: "   ", Date: someDate}, []error{ErrEmptyCategory}},
		{"нет даты", Transaction{Amount: 100, Category: "Еда"}, []error{ErrMissingDate}},
		{"всё сразу", Transaction{}, []error{ErrInvalidAmount, ErrEmptyCategory, ErrMissingDate}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertValidation(t, c.tx.Validate(), c.want)
		})
	}
}

func TestBudgetValidate(t *testing.T) {
	cases := []struct {
		name string
		b    Budget
		want []error
	}{
		{"корректный", Budget{Category: "Еда", Limit: 5000}, nil},
		{"нулевой лимит", Budget{Category: "Еда", Limit: 0}, []error{ErrInvalidLimit}},
		{"отрицательный лимит", Budget{Category: "Еда", Limit: -1}, []error{ErrInvalidLimit}},
		{"NaN", Budget{Category: "Еда", Limit: math.NaN()}, []error{ErrInvalidLimit}},
		{"пустая категория", Budget{Category: " ", Limit: 100}, []error{ErrEmptyCategory}},
		{"всё сразу", Budget{}, []error{ErrEmptyCategory, ErrInvalidLimit}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertValidation(t, c.b.Validate(), c.want)
		})
	}
}

// assertValidation проверяет, что err == nil при пустом want,
// а иначе — что это *ValidationError, содержащая ровно перечисленные проблемы.
func assertValidation(t *testing.T, err error, want []error) {
	t.Helper()
	if len(want) == 0 {
		if err != nil {
			t.Fatalf("ожидался nil, получено %v", err)
		}
		return
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("ожидалась *ValidationError, получено %T: %v", err, err)
	}
	if len(ve.Problems) != len(want) {
		t.Errorf("ожидалось проблем: %d, получено %d: %v", len(want), len(ve.Problems), err)
	}
	for _, w := range want {
		if !errors.Is(err, w) {
			t.Errorf("ошибка %q не содержит %q", err, w)
		}
	}
}

func TestCheckValidPolymorphism(t *testing.T) {
	cases := []struct {
		name    string
		v       Validatable
		wantErr error
	}{
		{"корректная транзакция", Transaction{Amount: 1, Category: "Еда", Date: someDate}, nil},
		{"некорректная транзакция", Transaction{Amount: -1, Category: "Еда", Date: someDate}, ErrInvalidAmount},
		{"указатель на транзакцию", &Transaction{Amount: 1, Category: "Еда", Date: someDate}, nil},
		{"корректный бюджет", Budget{Category: "Еда", Limit: 5000}, nil},
		{"некорректный бюджет", Budget{Category: "Еда", Limit: 0}, ErrInvalidLimit},
		{"указатель на бюджет", &Budget{Category: "", Limit: 1}, ErrEmptyCategory},
		{"nil", nil, ErrNilValidatable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckValid(c.v)
			if c.wantErr == nil && err != nil {
				t.Fatalf("ожидался nil, получено %v", err)
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Fatalf("ожидалась %v, получено %v", c.wantErr, err)
			}
		})
	}
}

func TestValidationErrorMessage(t *testing.T) {
	err := Transaction{Amount: -5}.Validate()
	want := "некорректная транзакция: сумма должна быть больше 0 (получено -5); категория не может быть пустой; дата не указана"
	if err == nil || err.Error() != want {
		t.Errorf("сообщение:\n  %v\nожидалось:\n  %s", err, want)
	}
}

// Валидация срабатывает раньше проверки бюджета, и некорректные объекты не сохраняются.
func TestAddUsesValidate(t *testing.T) {
	reset()
	if err := SetBudget(Budget{Category: "", Limit: -1}); err == nil {
		t.Fatal("некорректный бюджет должен быть отклонён")
	}
	if len(ListBudgets()) != 0 {
		t.Error("некорректный бюджет не должен сохраняться")
	}

	mustSetBudget(t, Budget{Category: "Еда", Limit: 100})
	err := AddTransaction(Transaction{Amount: -500, Category: "Еда"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("ожидалась ошибка валидации, получено %v", err)
	}
	if errors.Is(err, ErrBudgetExceeded) {
		t.Error("ошибка валидации не должна смешиваться с превышением бюджета")
	}
	if len(ListTransactions()) != 0 {
		t.Error("некорректная транзакция не должна сохраняться")
	}
}
