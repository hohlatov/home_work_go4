// Package ledger содержит бизнес-логику учёта транзакций и бюджетов.
//
// Файлы пакета:
//   - validation.go  — интерфейс Validatable, CheckValid, ошибки валидации;
//   - transaction.go — модель Transaction и её Validate;
//   - budget.go      — модель Budget и её Validate;
//   - ledger.go      — хранилище в памяти и бизнес-операции (этот файл);
//   - loader.go      — загрузка бюджетов из JSON.
package ledger

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrBudgetExceeded — транзакция нарушает бюджет категории.
var ErrBudgetExceeded = errors.New("budget exceeded")

// Хранилище в памяти. Транзакции и бюджеты защищены одним мьютексом,
// чтобы проверка бюджета и добавление транзакции выполнялись атомарно.
var (
	mu           sync.RWMutex
	transactions = make([]Transaction, 0)
	budgets      = make(map[string]Budget) // ключ — categoryKey(Category)
)

// categoryKey приводит название категории к единому виду,
// чтобы «Еда», «еда» и « еда » считались одной категорией.
func categoryKey(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}

// AddTransaction валидирует транзакцию, проверяет бюджет и добавляет её в хранилище.
// ID назначается автоматически (len(transactions)+1).
// Если дата не указана, подставляется текущее время.
//
// Возвращает *ValidationError, если данные некорректны, или ошибку,
// оборачивающую ErrBudgetExceeded, если транзакция превышает бюджет категории.
// В обоих случаях транзакция не сохраняется.
func AddTransaction(tx Transaction) error {
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}
	// Проверка данных отделена от бизнес-логики: модель проверяет себя сама.
	if err := tx.Validate(); err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	// Бизнес-правило: сумма по категории не должна превышать бюджет.
	if b, ok := budgets[categoryKey(tx.Category)]; ok {
		spent := spentLocked(tx.Category)
		if exceedsLimit(spent+tx.Amount, b.Limit) {
			return fmt.Errorf("%w: категория %q, лимит %.2f, уже потрачено %.2f, транзакция %.2f (доступно %.2f)",
				ErrBudgetExceeded, b.Category, b.Limit, spent, tx.Amount, b.Limit-spent)
		}
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

// ListTransactions возвращает копию всех сохранённых транзакций,
// чтобы вызывающий код не мог изменить внутреннее хранилище.
func ListTransactions() []Transaction {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}

// SpentByCategory возвращает сумму всех транзакций категории.
func SpentByCategory(category string) float64 {
	mu.RLock()
	defer mu.RUnlock()
	return spentLocked(category)
}

// SetBudget валидирует бюджет, затем добавляет его или обновляет лимит существующего.
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	b.Category = strings.TrimSpace(b.Category)

	mu.Lock()
	defer mu.Unlock()
	budgets[categoryKey(b.Category)] = b
	return nil
}

// GetBudget возвращает бюджет категории и признак его наличия.
func GetBudget(category string) (Budget, bool) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := budgets[categoryKey(category)]
	return b, ok
}

// ListBudgets возвращает все бюджеты, отсортированные по категории.
func ListBudgets() []Budget {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Budget, 0, len(budgets))
	for _, b := range budgets {
		result = append(result, b)
	}
	sort.Slice(result, func(i, j int) bool {
		return categoryKey(result[i].Category) < categoryKey(result[j].Category)
	})
	return result
}

// exceedsLimit сравнивает суммы с точностью до копеек. Так погрешность float64
// не помешает уложиться ровно в лимит (например, 0.1 + 0.2 при лимите 0.3).
func exceedsLimit(total, limit float64) bool {
	return math.Round(total*100) > math.Round(limit*100)
}

// spentLocked считает сумму транзакций категории. Вызывающий код должен держать mu.
func spentLocked(category string) float64 {
	key := categoryKey(category)
	var total float64
	for _, tx := range transactions {
		if categoryKey(tx.Category) == key {
			total += tx.Amount
		}
	}
	return total
}
