// Package main содержит комплексный мультичекер для статического анализа Go кода.
//
// StaticLint объединяет множество анализаторов для обнаружения потенциальных проблем,
// багов и нарушений стиля в Go коде. Включает в себя:
//
// - Стандартные анализаторы golang.org/x/tools/go/analysis/passes
// - Все анализаторы класса SA пакета staticcheck.io
// - Анализаторы других классов staticcheck.io (ST, S, QF)
// - Внешние публичные анализаторы (errcheck, ineffassign)
// - Собственный анализатор noosexit
//
// Использование:
//
//	staticlint ./...                    # анализ всех пакетов
//	staticlint .                        # анализ текущего пакета
//	staticlint -json ./...              # вывод в JSON формате
//	staticlint -c 10 ./...              # ограничить количество проблем
//
// Для подавления предупреждений используйте комментарии:
//
//	//lint:ignore SA1000 обоснованное исключение
//	problematicCode()
//
// Собственный анализатор noosexit запрещает прямые вызовы os.Exit в функции main
// пакета main, что улучшает тестируемость кода.
package main

import (
	"flag"
	"fmt"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"

	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/kisielk/errcheck/errcheck"

	"github.com/AlenaMolokova/http/cmd/staticlint/noosexit"
)

var (
	// version содержит версию multichecker
	version = "1.0.0"

	// showVersion флаг для показа версии
	showVersion = flag.Bool("version", false, "показать версию и выйти")

	// listAnalyzers флаг для показа списка анализаторов
	listAnalyzers = flag.Bool("list", false, "показать список всех анализаторов")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("staticlint version %s\n", version)
		return
	}

	analyzers := collectAnalyzers()

	if *listAnalyzers {
		printAnalyzersList(analyzers)
		return
	}

	multichecker.Main(analyzers...)
}

// collectAnalyzers собирает все анализаторы в одну коллекцию.
func collectAnalyzers() []*analysis.Analyzer {
	var analyzers []*analysis.Analyzer

	// Добавляем стандартные анализаторы
	analyzers = append(analyzers, getStandardAnalyzers()...)

	// Добавляем анализаторы StaticCheck
	analyzers = append(analyzers, getStaticCheckAnalyzers()...)

	// Добавляем анализаторы StyleCheck
	analyzers = append(analyzers, getStyleCheckAnalyzers()...)

	// Добавляем анализаторы Simple
	analyzers = append(analyzers, getSimpleAnalyzers()...)

	// Добавляем анализаторы QuickFix
	analyzers = append(analyzers, getQuickFixAnalyzers()...)

	// Добавляем внешние анализаторы
	analyzers = append(analyzers, getExternalAnalyzers()...)

	// Добавляем собственные анализаторы
	analyzers = append(analyzers, getCustomAnalyzers()...)

	return analyzers
}

// getStandardAnalyzers возвращает стандартные анализаторы Go.
func getStandardAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		asmdecl.Analyzer,      // проверяет соответствие объявлений ассемблера и Go
		assign.Analyzer,       // находит бесполезные присваивания
		atomic.Analyzer,       // проверяет ошибки при использовании sync/atomic
		bools.Analyzer,        // проверяет ошибки с булевыми операторами
		buildtag.Analyzer,     // проверяет синтаксис build-тегов
		cgocall.Analyzer,      // проверяет нарушения правил cgo pointer passing
		composite.Analyzer,    // проверяет неинициализированные поля в композитных литералах
		copylock.Analyzer,     // проверяет блокировки, переданные по значению
		errorsas.Analyzer,     // проверяет неправильное использование errors.As
		httpresponse.Analyzer, // проверяет ошибки при использовании HTTP ответов
		loopclosure.Analyzer,  // проверяет ссылки на переменные цикла из вложенных функций
		lostcancel.Analyzer,   // проверяет неиспользуемые результаты context.WithCancel
		nilfunc.Analyzer,      // проверяет бесполезные сравнения func == nil
		printf.Analyzer,       // проверяет соответствие Printf-подобных функций
		shift.Analyzer,        // проверяет сдвиги, равные или превышающие ширину целого числа
		stdmethods.Analyzer,   // проверяет подписи методов известных интерфейсов
		structtag.Analyzer,    // проверяет теги структур
		tests.Analyzer,        // проверяет ошибочные использования тестов и примеров
		unmarshal.Analyzer,    // проверяет передачу не-указателей в функции unmarshal
		unreachable.Analyzer,  // находит недостижимый код
		unsafeptr.Analyzer,    // проверяет недопустимые преобразования uintptr в unsafe.Pointer
		unusedresult.Analyzer, // проверяет неиспользуемые результаты вызовов определенных функций
	}
}

// getStaticCheckAnalyzers возвращает все анализаторы класса SA из staticcheck.
func getStaticCheckAnalyzers() []*analysis.Analyzer {
	var analyzers []*analysis.Analyzer

	// Добавляем все анализаторы SA из staticcheck
	for _, analyzer := range staticcheck.Analyzers {
		if strings.HasPrefix(analyzer.Analyzer.Name, "SA") {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	return analyzers
}

// getStyleCheckAnalyzers возвращает анализаторы класса ST из stylecheck.
func getStyleCheckAnalyzers() []*analysis.Analyzer {
	var analyzers []*analysis.Analyzer

	// Добавляем анализаторы ST из stylecheck
	for _, analyzer := range stylecheck.Analyzers {
		if strings.HasPrefix(analyzer.Analyzer.Name, "ST") {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	return analyzers
}

// getSimpleAnalyzers возвращает анализаторы класса S из simple.
func getSimpleAnalyzers() []*analysis.Analyzer {
	var analyzers []*analysis.Analyzer

	// Добавляем анализаторы S из simple
	for _, analyzer := range simple.Analyzers {
		if strings.HasPrefix(analyzer.Analyzer.Name, "S") {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	return analyzers
}

// getQuickFixAnalyzers возвращает анализаторы класса QF из quickfix.
func getQuickFixAnalyzers() []*analysis.Analyzer {
	var analyzers []*analysis.Analyzer

	// Добавляем анализаторы QF из quickfix
	for _, analyzer := range quickfix.Analyzers {
		if strings.HasPrefix(analyzer.Analyzer.Name, "QF") {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	return analyzers
}

// getExternalAnalyzers возвращает внешние публичные анализаторы.
func getExternalAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		errcheck.Analyzer,    // проверяет неиспользуемые ошибки
		ineffassign.Analyzer, // находит бесполезные присваивания
	}
}

// getCustomAnalyzers возвращает собственные анализаторы.
func getCustomAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		noosexit.Analyzer, // запрещает прямые вызовы os.Exit в функции main пакета main
	}
}

// printAnalyzersList выводит список всех доступных анализаторов.
func printAnalyzersList(analyzers []*analysis.Analyzer) {
	fmt.Printf("StaticLint v%s - доступные анализаторы:\n\n", version)

	// Группируем анализаторы по типам
	groups := map[string][]*analysis.Analyzer{
		"Стандартные анализаторы": {},
		"StaticCheck (SA)":    {},
		"StyleCheck (ST)":     {},
		"Simple (S)":          {},
		"QuickFix (QF)":       {},
		"Внешние анализаторы": {},
		"Собственные анализаторы": {},
	}

	for _, analyzer := range analyzers {
		name := analyzer.Name
		switch {
		case strings.HasPrefix(name, "SA"):
			groups["StaticCheck (SA)"] = append(groups["StaticCheck (SA)"], analyzer)
		case strings.HasPrefix(name, "ST"):
			groups["StyleCheck (ST)"] = append(groups["StyleCheck (ST)"], analyzer)
		case strings.HasPrefix(name, "S"):
			groups["Simple (S)"] = append(groups["Simple (S)"], analyzer)
		case strings.HasPrefix(name, "QF"):
			groups["QuickFix (QF)"] = append(groups["QuickFix (QF)"], analyzer)
		case name == "errcheck" || name == "ineffassign":
			groups["Внешние анализаторы"] = append(groups["Внешние анализаторы"], analyzer)
		case name == "noosexit":
			groups["Собственные анализаторы"] = append(groups["Собственные анализаторы"], analyzer)
		default:
			groups["Стандартные анализаторы"] = append(groups["Стандартные анализаторы"], analyzer)
		}
	}

	// Выводим группы
	for groupName, groupAnalyzers := range groups {
		if len(groupAnalyzers) == 0 {
			continue
		}

		fmt.Printf("=== %s ===\n", groupName)
		for _, analyzer := range groupAnalyzers {
			fmt.Printf("  %-15s %s\n", analyzer.Name, getShortDoc(analyzer.Doc))
		}
		fmt.Println()
	}

	fmt.Printf("Всего анализаторов: %d\n", len(analyzers))
}

// getShortDoc возвращает краткое описание анализатора.
func getShortDoc(doc string) string {
	lines := strings.Split(doc, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		short := strings.TrimSpace(lines[0])
		if len(short) > 80 {
			return short[:77] + "..."
		}
		return short
	}
	return "описание отсутствует"
}
