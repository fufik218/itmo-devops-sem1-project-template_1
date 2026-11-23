package main

import (
	"archive/zip"           // работа с zip-архивами
	"bytes"                 // буфер в памяти для чтения/записи архива
	"database/sql"          // интерфейс к БД
	"encoding/csv"          // парсинг и запись CSV
	"encoding/json"         // формирование JSON-ответов
	"fmt"                   // форматированный вывод
	"io"                    // чтение тела запроса
	"log"                   // логирование
	"net/http"              // HTTP-сервер
	"strconv"               // преобразование строк в числа
	"strings"               // работа со строками

	_ "github.com/lib/pq"   // драйвер PostgreSQL (blank import для регистрации)
)

// PostResponse — структура JSON-ответа на POST /api/v0/prices
type PostResponse struct {
	TotalItems      int     `json:"total_items"`      // общее количество добавленных элементов
	TotalCategories int     `json:"total_categories"` // количество уникальных категорий
	TotalPrice      float64 `json:"total_price"`      // суммарная стоимость (float, т.к. цены дробные)
}

// connStr — строка подключения к PostgreSQL (параметры из чек-листа)
const connStr = "user=validator password=val1dat0r dbname=project-sem-1 host=localhost port=5432 sslmode=disable"

func main() {
	// 1. Подключение к БД
	db, err := sql.Open("postgres", connStr) // открываем пул соединений
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err) // фатальная ошибка — выходим
	}
	defer db.Close() // закрываем пул при завершении программы

	// 2. Проверка, что БД реально доступна
	if err := db.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err) // если ping не прошёл — выходим
	}

	// 3. Регистрация HTTP-обработчика для /api/v0/prices
	http.HandleFunc("/api/v0/prices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method { // роутинг по HTTP-методу
		case http.MethodPost:
			handlePost(db, w, r) // обработка POST
		case http.MethodGet:
			handleGet(db, w, r) // обработка GET
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // остальные методы — 405
		}
	})

	// 4. Запуск сервера на порту 8080
	fmt.Println("Сервер слушает :8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) // блокирующий вызов
}

// handlePost — обработчик POST /api/v0/prices
func handlePost(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	// 1. Читаем всё тело запроса (zip-архив)
	body, err := io.ReadAll(r.Body) // читаем байты
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest) // 400
		return
	}
	defer r.Body.Close() // закрываем тело

	// 2. Открываем zip-архив в памяти
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body))) // создаём ридер zip
	if err != nil {
		http.Error(w, "Неверный zip-архив", http.StatusBadRequest) // 400
		return
	}

	// Переменные для подсчёта статистики
	var totalItems int
	categoriesMap := make(map[string]bool) // множество уникальных категорий
	var totalPrice float64

	// 3. Перебираем все файлы внутри архива
	for _, file := range zipReader.File {
		if !strings.HasSuffix(file.Name, ".csv") { // ищем только CSV-файлы
			continue // остальные пропускаем
		}

		// Открываем CSV-файл внутри архива
		rc, err := file.Open()
		if err != nil {
			http.Error(w, "Ошибка чтения файла из архива", http.StatusInternalServerError) // 500
			return
		}
		defer rc.Close() // закрываем файл

		// 4. Читаем CSV
		reader := csv.NewReader(rc)
		records, err := reader.ReadAll() // читаем все строки
		if err != nil {
			http.Error(w, "Ошибка парсинга CSV", http.StatusInternalServerError) // 500
			return
		}

		// 5. Обрабатываем каждую строку (первая — заголовок)
		for i, row := range records {
			if i == 0 { // заголовок: id,name,category,price,create_date
				continue // пропускаем
			}
			if len(row) < 4 { // минимум 4 колонки: id, name, category, price
				continue // если меньше — пропускаем строку
			}

			name := row[1]     // колонка name
			category := row[2] // колонка category

			// Цена — дробная (799.99), парсим как float64
			price, err := strconv.ParseFloat(row[3], 64)
			if err != nil {
				continue // если не число — пропускаем строку
			}

			// 6. Вставляем строку в БД (параметризованный запрос — защита от SQL-инъекций)
			_, err = db.Exec("INSERT INTO prices (name, category, price) VALUES ($1, $2, $3)", name, category, price)
			if err != nil {
				log.Printf("Ошибка вставки: %v", err) // логируем, но не прерываем обработку
				continue
			}

			// 7. Обновляем статистику
			totalItems++                   // +1 элемент
			categoriesMap[category] = true // категория в множество (уникальность)
			totalPrice += price            // суммируем цену
		}
	}

	// 8. Формируем JSON-ответ
	response := PostResponse{
		TotalItems:      totalItems,
		TotalCategories: len(categoriesMap), // количество уникальных категорий
		TotalPrice:      totalPrice,
	}

	w.Header().Set("Content-Type", "application/json") // заголовок JSON
	json.NewEncoder(w).Encode(response)               // кодируем и пишем ответ
}

// handleGet — обработчик GET /api/v0/prices
func handleGet(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	// 1. Запрашиваем все записи из БД
	rows, err := db.Query("SELECT name, category, price FROM prices")
	if err != nil {
		http.Error(w, "Ошибка запроса к БД", http.StatusInternalServerError) // 500
		return
	}
	defer rows.Close() // закрываем rows

	// 2. Создаём zip-архив в памяти
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf) // writer zip

	// 3. Создаём файл data.csv внутри архива
	csvFile, err := zipWriter.Create("data.csv")
	if err != nil {
		http.Error(w, "Ошибка создания zip", http.StatusInternalServerError) // 500
		return
	}

	// 4. Пишем CSV
	csvWriter := csv.NewWriter(csvFile)
	csvWriter.Write([]string{"name", "category", "price"}) // заголовок CSV

	// 5. Перебираем строки из БД и пишем в CSV
	for rows.Next() {
		var name, category string
		var price float64

		if err := rows.Scan(&name, &category, &price); err != nil {
			continue // если ошибка — пропускаем строку
		}

		priceStr := strconv.FormatFloat(price, 'f', 2, 64) // форматируем цену с 2 знаками
		csvWriter.Write([]string{name, category, priceStr})
	}

	// 6. Завершаем запись
	csvWriter.Flush() // сбрасываем CSV-буфер
	zipWriter.Close() // закрываем zip

	// 7. Отправляем архив клиенту
	w.Header().Set("Content-Type", "application/zip")                       // тип содержимого
	w.Header().Set("Content-Disposition", "attachment; filename=data.zip") // скачивание
	w.Write(buf.Bytes())                                                   // пишем байты архива
}
