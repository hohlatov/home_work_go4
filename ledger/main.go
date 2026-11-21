package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"ledger/internal/ledger"
)

func main() {
	budgetsPath := flag.String("budgets", "budgets.json", "путь к JSON-файлу с бюджетами")
	flag.Parse()

	fmt.Println("Ledger service started")

	// 1. Начальные бюджеты задаются через SetBudget.
	section("1. Начальные бюджеты (SetBudget)")
	for _, b := range []ledger.Budget{
		{Category: "Еда", Limit: 5000},
		{Category: "Транспорт", Limit: 3000},
	} {
		if err := ledger.SetBudget(b); err != nil {
			log.Fatalf("не удалось установить бюджет: %v", err)
		}
		fmt.Printf("установлен бюджет: %-12s %10.2f\n", b.Category, b.Limit)
	}

	// 2. Загрузка бюджетов из файла (os.Open + bufio.NewReader).
	// Бюджет «Транспорт» из файла заменяет начальный.
	section("2. Загрузка бюджетов из " + *budgetsPath)
	if err := loadBudgetsFromFile(*budgetsPath); err != nil {
		log.Fatalf("ошибка: %v", err)
	}
	fmt.Println("бюджеты загружены")
	printBudgets()

	// 3. Как обрабатываются некорректные входные данные.
	section("3. Ошибки загрузки бюджетов")
	if err := loadBudgetsFromFile("missing.json"); err != nil {
		fmt.Println("ошибка:", err)
	}
	for _, bad := range []string{
		`[{"category": "Еда", "limit": 5000,}]`,
		`[{"category": "Еда", "limit": "много"}]`,
		`[{"category": "Еда", "limt": 5000}]`,
		`[{"category": "Еда", "limit": -100}]`,
		`{"category": "Еда", "limit": 5000}`,
	} {
		if err := ledger.LoadBudgets(strings.NewReader(bad)); err != nil {
			fmt.Printf("%-42s -> %v\n", bad, err)
		}
	}

	// 4. Полиморфизм: CheckValid принимает любой Validatable — и транзакцию, и бюджет.
	section("4. Проверка через интерфейс Validatable (CheckValid)")
	objects := []ledger.Validatable{
		tx(1250.50, "Еда", "Корректная транзакция", "2026-11-02"),
		ledger.Transaction{Amount: -300, Category: "", Description: "Всё неправильно"},
		ledger.Budget{Category: "Еда", Limit: 5000},
		ledger.Budget{Category: "Подписки", Limit: 0},
		&ledger.Budget{Category: "   ", Limit: -10},
	}
	for _, obj := range objects {
		// Validate ничего не логирует, ошибку выводит вызывающий код.
		if err := ledger.CheckValid(obj); err != nil {
			fmt.Printf("✗ %-20T %v\n", obj, err)
		} else {
			fmt.Printf("✓ %-20T корректно\n", obj)
		}
	}

	// 5. Сценарии добавления транзакций (Validate вызывается внутри AddTransaction).
	section("5. Добавление транзакций")
	scenarios := []struct {
		tx      ledger.Transaction
		wantErr error // nil — ожидается успех
	}{
		{tx(1250.50, "Еда", "Покупки в супермаркете", "2026-11-02"), nil},
		{tx(3200, "Еда", "Закупка на неделю", "2026-11-09"), nil},
		{tx(800, "Еда", "Ресторан (превышение бюджета)", "2026-11-12"), ledger.ErrBudgetExceeded},
		{tx(549.50, "еда", "Ровно до лимита", "2026-11-14"), nil},
		{tx(350, "Транспорт", "Такси до университета", "2026-11-03"), nil},
		{tx(1800, "Транспорт", "Проездной (превышение бюджета)", "2026-11-05"), ledger.ErrBudgetExceeded},
		{tx(4990, "Образование", "Подписка на курс по Go", "2026-11-10"), nil},
		{tx(10000, "Подарки", "Категория без бюджета", "2026-11-15"), nil},
		{tx(0, "Прочее", "Нулевая сумма", "2026-11-16"), ledger.ErrInvalidAmount},
		{tx(-200, "Еда", "Отрицательная сумма", "2026-11-16"), ledger.ErrInvalidAmount},
		{tx(150, "  ", "Пустая категория", "2026-11-16"), ledger.ErrEmptyCategory},
	}

	accepted := 0
	for _, s := range scenarios {
		if addAndReport(s.tx, s.wantErr) {
			accepted++
		}
	}

	// 6. Увеличиваем лимит через SetBudget и повторяем отклонённую транзакцию.
	section("6. Обновление бюджета и повторная попытка")
	if err := ledger.SetBudget(ledger.Budget{Category: "Транспорт", Limit: 2500}); err != nil {
		log.Fatalf("не удалось обновить бюджет: %v", err)
	}
	fmt.Println("лимит «Транспорт» увеличен до 2500.00")
	if addAndReport(tx(1800, "Транспорт", "Проездной (после увеличения лимита)", "2026-11-05"), nil) {
		accepted++
	}

	// 7. Итог.
	section("7. Сохранённые транзакции")
	list := ledger.ListTransactions()
	fmt.Printf("%-4s %-12s %-12s %10s  %s\n", "ID", "Дата", "Категория", "Сумма", "Описание")
	for _, t := range list {
		fmt.Printf("%-4d %-12s %-12s %10.2f  %s\n",
			t.ID, t.Date.Format("2006-01-02"), t.Category, t.Amount, t.Description)
	}

	section("8. Исполнение бюджетов")
	printBudgets()

	fmt.Println()
	if len(list) != accepted {
		log.Fatalf("ОШИБКА: принято %d транзакций, а в хранилище %d", accepted, len(list))
	}
	fmt.Printf("Проверка пройдена: в хранилище %d транзакций, все отклонённые отсутствуют.\n", len(list))
}

// loadBudgetsFromFile открывает файл и передаёт буферизованный reader в LoadBudgets.
func loadBudgetsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл бюджетов %q: %w", path, err)
	}
	defer f.Close()

	if err := ledger.LoadBudgets(bufio.NewReader(f)); err != nil {
		return fmt.Errorf("файл %q: %w", path, err)
	}
	return nil
}

// addAndReport добавляет транзакцию, печатает результат и сверяет его с ожидаемым.
// Возвращает true, если транзакция сохранена.
func addAndReport(t ledger.Transaction, wantErr error) bool {
	err := ledger.AddTransaction(t)

	status := "✓"
	if (wantErr == nil) != (err == nil) || (wantErr != nil && !errors.Is(err, wantErr)) {
		status = "✗ НЕОЖИДАННО"
	}

	if err != nil {
		fmt.Printf("%s ОТКАЗ   %-12s %9.2f  %s\n           причина: %v\n",
			status, t.Category, t.Amount, t.Description, err)
		return false
	}
	fmt.Printf("%s ПРИНЯТО %-12s %9.2f  %s\n", status, t.Category, t.Amount, t.Description)
	return true
}

func printBudgets() {
	fmt.Printf("%-12s %10s %10s %10s\n", "Категория", "Лимит", "Потрачено", "Остаток")
	for _, b := range ledger.ListBudgets() {
		spent := ledger.SpentByCategory(b.Category)
		fmt.Printf("%-12s %10.2f %10.2f %10.2f\n", b.Category, b.Limit, spent, b.Limit-spent)
	}
}

func section(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

// tx — короткий конструктор транзакции для демонстрации.
func tx(amount float64, category, description, date string) ledger.Transaction {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatalf("некорректная дата %q: %v", date, err)
	}
	return ledger.Transaction{Amount: amount, Category: category, Description: description, Date: d}
}
