package ledger

import (
	"errors"
	"testing"
)

func reset() {
	mu.Lock()
	transactions = make([]Transaction, 0)
	budgets = make(map[string]Budget)
	mu.Unlock()
}

func TestAddTransaction(t *testing.T) {
	reset()

	if err := AddTransaction(Transaction{Amount: 100, Category: "Еда"}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if err := AddTransaction(Transaction{Amount: 250.5, Category: "Транспорт"}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	list := ListTransactions()
	if len(list) != 2 {
		t.Fatalf("ожидалось 2 транзакции, получено %d", len(list))
	}
	for i, tx := range list {
		if tx.ID != i+1 {
			t.Errorf("транзакция %d: ожидался ID %d, получен %d", i, i+1, tx.ID)
		}
		if tx.Date.IsZero() {
			t.Errorf("транзакция %d: дата не заполнена", i)
		}
	}
}

func TestAddTransactionValidation(t *testing.T) {
	reset()

	if err := AddTransaction(Transaction{Amount: 0, Category: "Еда"}); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("ожидалась ErrInvalidAmount, получено %v", err)
	}
	if err := AddTransaction(Transaction{Amount: -50, Category: "Еда"}); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("отрицательная сумма: ожидалась ErrInvalidAmount, получено %v", err)
	}
	if err := AddTransaction(Transaction{Amount: 10, Category: "  "}); !errors.Is(err, ErrEmptyCategory) {
		t.Errorf("ожидалась ErrEmptyCategory, получено %v", err)
	}
	if n := len(ListTransactions()); n != 0 {
		t.Errorf("невалидные транзакции не должны сохраняться, в хранилище %d", n)
	}
}

func TestListTransactionsReturnsCopy(t *testing.T) {
	reset()
	_ = AddTransaction(Transaction{Amount: 1, Category: "Тест"})

	list := ListTransactions()
	list[0].Amount = 999

	if got := ListTransactions()[0].Amount; got != 1 {
		t.Errorf("изменение копии затронуло хранилище: Amount = %v", got)
	}
}
