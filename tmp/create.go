package createPDF

//
//import (
//	"bytes"
//	"context"
//	"fmt"
//	createpdffile "github.com/arsteldev/createPDF/proto"
//	"github.com/jung-kurt/gofpdf"
//	"log"
//	"log/slog"
//	"os"
//	"path/filepath"
//	"strconv"
//	"time"
//)
//
//type PDFServer struct {
//	createpdffile.UnimplementedPDFCreatorServer
//	Log *slog.Logger
//}
//
//func (s *PDFServer) CreatePDF(ctx context.Context, req *createpdffile.CreatePDFRequest) (*createpdffile.CreatePDFResponse, error) {
//	s.Log.Info("Received request from email: %s, phone: %s",
//		slog.String("email", req.GetEmail()),
//		slog.String("phone", req.GetPhone()))
//
//	s.Log.Info("Создание PDF файла")
//
//	pdfBytes, err := s.generatePDF(req)
//	s.Log.Info("Успешное создание PDF файла")
//
//	if err != nil {
//		return &createpdffile.CreatePDFResponse{
//			Success:      false,
//			ErrorMessage: err.Error(),
//		}, nil
//	}
//
//	return &createpdffile.CreatePDFResponse{
//		Pdf:     pdfBytes,
//		Success: true,
//	}, nil
//}
//
//func (s *PDFServer) generatePDF(req *createpdffile.CreatePDFRequest) ([]byte, error) {
//	fontDir := "/var/www/createpdf/font/"
//
//	pdf := gofpdf.NewCustom(&gofpdf.InitType{
//		OrientationStr: "L",
//		UnitStr:        "mm",
//		SizeStr:        "A4",
//		FontDirStr:     fontDir,
//	})
//
//	pdf.SetAutoPageBreak(true, 10)
//
//	// Проверяемые файлы шрифтов
//	fontFiles := map[string]string{
//		"regular": "DejaVuSans.ttf",
//		"bold":    "DejaVuSans-Bold.ttf",
//		"italic":  "DejaVuSans-Oblique.ttf",
//	}
//
//	// Проверяем существование шрифтов
//	for name, filename := range fontFiles {
//		fullPath := fontDir + filename
//		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
//			s.Log.Error("Файл шрифта не найден",
//				slog.String("name", name),
//				slog.String("path", fullPath))
//			return nil, fmt.Errorf("файл шрифта не найден: %s", fullPath)
//		}
//		s.Log.Info("Шрифт доступен",
//			slog.String("name", name),
//			slog.String("file", filename))
//	}
//
//	// Добавляем UTF-8 шрифты (только имена файлов)
//	//pdf.AddUTF8Font("dejavu", "", "DejaVuSans.ttf")
//	//pdf.AddUTF8Font("dejavu", "B", "DejaVuSans-Bold.ttf")
//	//pdf.AddUTF8Font("dejavu", "I", "DejaVuSans-Oblique.ttf")
//
//	pdf.AddUTF8Font("montserrat", "", "Montserrat-Regular.ttf")
//	pdf.AddUTF8Font("montserrat", "B", "Montserrat-Bold.ttf")
//	pdf.AddUTF8Font("montserrat", "I", "Montserrat-Italic.ttf")
//
//	// Устанавливаем шрифт по умолчанию
//	pdf.SetFont("montserrat", "", 12)
//
//	// Устанавливаем автора и заголовок
//	pdf.SetAuthor(req.GetEmail(), true)
//	pdf.SetTitle("Отчет по моделям", true)
//
//	// Страница 1: Заставка с ФИО и номером
//	createCoverPage(pdf, req.GetEmail(), req.GetPhone())
//
//	// Страница 2: Содержание
//	createTableOfContents(pdf)
//
//	// Страница 3: Таблица с model_id и картинками
//	createModelTable(pdf, req.GetModels())
//
//	// Страница 4: Заключительная страница
//	createThankYouPage(pdf)
//
//	// ВАРИАНТ 1: Сохраняем в файл И возвращаем байты
//	var buf bytes.Buffer
//	err := pdf.Output(&buf)
//	if err != nil {
//		return nil, fmt.Errorf("ошибка создания буфера PDF: %v", err)
//	}
//
//	// Дополнительно сохраняем в файл (если нужно)
//	exePath, err := os.Executable()
//	if err != nil {
//		return nil, fmt.Errorf("ошибка получения пути исполняемого файла: %v", err)
//	}
//	outputDir := filepath.Dir(exePath)
//
//	log.Printf("outputDir: %s", outputDir)
//
//	timestamp := time.Now().Format("20060102_150405")
//	filename := fmt.Sprintf("report_%s.pdf", timestamp)
//	fullPath := filepath.Join(outputDir, filename)
//
//	// Сохраняем буфер в файл
//	err = os.WriteFile(fullPath, buf.Bytes(), 0644)
//	if err != nil {
//		log.Printf("Ошибка сохранения в файл: %v", err)
//	} else {
//		log.Printf("PDF файл успешно сохранен: %s", fullPath)
//	}
//
//	return buf.Bytes(), nil
//}
//
//// Страница 1: Заставка
//func createCoverPage(pdf *gofpdf.Fpdf, manager, phone string) {
//	pdf.AddPage()
//
//	// Создаем цветной фон
//	pdf.SetFillColor(41, 128, 185) // Синий цвет
//	pdf.Rect(0, 0, 297, 210, "F")
//
//	// Заголовок
//	pdf.SetY(100)
//	pdf.SetFont("montserrat", "B", 24)
//	pdf.SetTextColor(255, 255, 255)
//	pdf.CellFormat(0, 20, "ОТЧЕТ ПО МОДЕЛЯМ", "", 0, "C", false, 0, "")
//
//	// Информация о менеджере
//	pdf.SetY(150)
//	pdf.SetFont("montserrat", "B", 16)
//	pdf.SetTextColor(255, 255, 255)
//	pdf.CellFormat(0, 10, "Почта: "+manager, "", 0, "C", false, 0, "")
//
//	// Номер телефона
//	pdf.SetY(170)
//	pdf.CellFormat(0, 10, "Телефон: "+phone, "", 0, "C", false, 0, "")
//
//	// Дата
//	pdf.SetY(250)
//	pdf.SetFont("montserrat", "I", 12)
//	pdf.CellFormat(0, 10, time.Now().Format("02.01.2006"), "", 0, "C", false, 0, "")
//}
//
//// Страница 2: Содержание (теперь знает номер страницы таблицы)
//func createTableOfContents(pdf *gofpdf.Fpdf, modelTableStartPage int) {
//	pdf.AddPage()
//
//	// Заголовок
//	pdf.SetFont("montserrat", "B", 20)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 20, "Содержание", "", 1, "C", false, 0, "")
//	pdf.Ln(10)
//
//	// Элементы содержания
//	pdf.SetFont("montserrat", "", 14)
//
//	// Таблица моделей
//	pdf.SetX(20)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 10, "Таблица моделей", "", 0, "L", false, 0, "")
//	pdf.SetX(170)
//	pdf.CellFormat(20, 10, fmt.Sprintf("%d", modelTableStartPage), "", 1, "R", false, 0, "")
//
//	// Заключение (автоматически рассчитываем на 1 страницу позже)
//	conclusionPage := modelTableStartPage + 1
//	// Подсчитываем сколько страниц займет таблица моделей
//	totalModels := len(pictures) // pictures должен быть доступен здесь
//	modelsOnFirstPage := 3
//	modelsOnOtherPages := 4
//
//	// Вычисляем сколько страниц займет таблица
//	remainingModels := totalModels - modelsOnFirstPage
//	if remainingModels > 0 {
//		extraPages := (remainingModels + modelsOnOtherPages - 1) / modelsOnOtherPages
//		conclusionPage = modelTableStartPage + extraPages + 1
//	}
//
//	pdf.SetX(20)
//	pdf.CellFormat(0, 10, "Заключение", "", 0, "L", false, 0, "")
//	pdf.SetX(170)
//	pdf.CellFormat(20, 10, fmt.Sprintf("%d", conclusionPage), "", 1, "R", false, 0, "")
//
//	pdf.Ln(20)
//
//	// Разделительная линия
//	pdf.SetDrawColor(200, 200, 200)
//	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
//}
//
//// Константы для таблицы моделей
//const (
//	rowHeight           = 40.0
//	imageSize           = 30.0
//	maxModelsFirstPage  = 3
//	maxModelsOtherPages = 4
//)
//
//// Ширины для альбомной ориентации
//var tableWidths = []float64{
//	20,  // №
//	110, // Наименование
//	30,  // Кол-во
//	35,  // Цена
//	35,  // Сумма
//	30,  // Наличие
//	25,  // Фото
//}
//
//// Функция для создания заголовка таблицы
//func createTableHeader(pdf *gofpdf.Fpdf) {
//	headerHeight := 12.0
//
//	// Сбрасываем Y позицию для начала таблицы
//	pdf.SetY(25) // Фиксированный отступ сверху
//
//	pdf.SetFillColor(200, 200, 200)
//	pdf.SetFont("montserrat", "B", 10)
//
//	// Заголовки таблицы
//	headers := []string{
//		"№",
//		"Наименование, описание оборудования",
//		"Кол-во, шт.",
//		"Цена, руб.",
//		"Сумма, руб.",
//		"Наличие",
//		"Фото",
//	}
//
//	// Устанавливаем левое поле для всей таблицы
//	pdf.SetLeftMargin(10)
//
//	// Рисуем ячейки заголовка
//	for i, header := range headers {
//		if i == len(headers)-1 {
//			pdf.CellFormat(tableWidths[i], headerHeight, header, "1", 1, "C", true, 0, "")
//		} else {
//			pdf.CellFormat(tableWidths[i], headerHeight, header, "1", 0, "C", true, 0, "")
//		}
//	}
//
//	pdf.SetFont("montserrat", "", 11)
//	// Маленький отступ после заголовка (уменьшен)
//	pdf.Ln(1)
//}
//
//func createModelTable(pdf *gofpdf.Fpdf, pictures []*createpdffile.Models) int {
//	modelsOnCurrentPage := 0
//	isFirstPage := true
//	pageNumber := 1 // Текущая страница таблицы (начинается с 1)
//
//	// Сбрасываем позицию и поля перед началом таблицы
//	pdf.SetLeftMargin(10)
//	pdf.SetRightMargin(10)
//	pdf.SetTopMargin(15)
//	pdf.SetY(25) // Начальная позиция Y
//
//	for i, picture := range pictures {
//		// Если это начало новой страницы
//		if modelsOnCurrentPage == 0 {
//			// Добавляем новую страницу, если это не первый элемент
//			if i > 0 {
//				pdf.AddPage()
//				pageNumber++
//			}
//
//			// На первой странице добавляем заголовок
//			if isFirstPage && pageNumber == 1 {
//				createTableHeader(pdf)
//			}
//			// На последующих страницах заголовка нет
//		}
//
//		// Определяем лимит для текущей страницы
//		maxModels := maxModelsOtherPages
//		if isFirstPage && pageNumber == 1 {
//			maxModels = maxModelsFirstPage
//		}
//
//		// Если достигли лимита на текущей странице - добавляем новую
//		if modelsOnCurrentPage >= maxModels {
//			pdf.AddPage()
//			pageNumber++
//			modelsOnCurrentPage = 0
//
//			// После первой страницы сбрасываем флаг
//			if isFirstPage {
//				isFirstPage = false
//			}
//
//			// Сбрасываем позицию на новой странице
//			pdf.SetY(15) // Отступ сверху на последующих страницах
//		}
//
//		// Чередование цвета фона
//		if modelsOnCurrentPage%2 == 0 {
//			pdf.SetFillColor(255, 255, 255) // Белый
//		} else {
//			pdf.SetFillColor(250, 250, 250) // Светло-серый
//		}
//
//		currentY := pdf.GetY()
//
//		// № строки
//		pdf.CellFormat(tableWidths[0], rowHeight, strconv.Itoa(i+1), "1", 0, "C", true, 0, "")
//
//		// Название модели (обрезаем если слишком длинное)
//		name := picture.GetName()
//		maxNameLength := 50
//		if len(name) > maxNameLength {
//			name = name[:maxNameLength] + "..."
//		}
//		pdf.CellFormat(tableWidths[1], rowHeight, name, "1", 0, "L", true, 0, "")
//
//		// Кол-во
//		pdf.CellFormat(tableWidths[2], rowHeight, strconv.Itoa(int(picture.Count)), "1", 0, "C", true, 0, "")
//
//		// Цена
//		pdf.CellFormat(tableWidths[3], rowHeight, "0", "1", 0, "C", true, 0, "")
//
//		// Сумма
//		pdf.CellFormat(tableWidths[4], rowHeight, "0", "1", 0, "C", true, 0, "")
//
//		// Наличие
//		pdf.CellFormat(tableWidths[5], rowHeight, "0", "1", 0, "C", true, 0, "")
//
//		// Ячейка для фото
//		xPos := pdf.GetX()
//		pdf.CellFormat(tableWidths[6], rowHeight, "", "1", 1, "C", true, 0, "")
//
//		// Добавление изображения
//		if imagePath := picture.GetImg(); imagePath != "" {
//			imageX := xPos + (tableWidths[6]-imageSize)/2
//			imageY := currentY + (rowHeight-imageSize)/2
//
//			if _, err := os.Stat(imagePath); err == nil {
//				// Пытаемся добавить изображение
//				pdf.Image(imagePath, imageX, imageY, imageSize, imageSize, false, "", 0, "")
//			} else {
//				// Если изображение не найдено - рисуем рамку
//				pdf.SetDrawColor(200, 200, 200)                      // Серый цвет рамки
//				pdf.SetFillColor(240, 240, 240)                      // Светло-серый фон
//				pdf.Rect(imageX, imageY, imageSize, imageSize, "FD") // F-заливка, D-рамка
//
//				// Текст "No Image"
//				pdf.SetTextColor(150, 150, 150) // Серый текст
//				pdf.SetFont("montserrat", "I", 6)
//				textX := imageX + (imageSize-pdf.GetStringWidth("No Image"))/2
//				textY := imageY + imageSize/2 + 2
//				pdf.Text(textX, textY, "No Image")
//				pdf.SetFont("montserrat", "", 11)
//				pdf.SetTextColor(0, 0, 0)
//			}
//		}
//
//		modelsOnCurrentPage++
//	}
//
//	return pageNumber
//}
//
//// Страница заключения
//func createThankYouPage(pdf *gofpdf.Fpdf) {
//	pdf.AddPage()
//
//	// Заголовок
//	pdf.SetFont("montserrat", "B", 24)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 100, "Заключение", "", 1, "C", false, 0, "")
//
//	// Текст заключения
//	pdf.SetFont("montserrat", "", 14)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.MultiCell(0, 10,
//		"Настоящий отчет содержит информацию о всех моделях, находящихся в базе данных. "+
//			"Каждая модель представлена с фотографией, техническими характеристиками и статусом наличия. "+
//			"Отчет был сгенерирован автоматически и содержит актуальную информацию на момент создания.",
//		"", "C", false)
//
//	pdf.Ln(20)
//
//	// Благодарность
//	pdf.SetFont("montserrat", "I", 12)
//	pdf.CellFormat(0, 10, "Благодарим за использование нашего сервиса!", "", 1, "C", false, 0, "")
//}
//
//// Основная функция генерации PDF
//func CreatePDF(manager, phone string, pictures []*createpdffile.Models, outputPath string) error {
//	// Создаем PDF с альбомной ориентацией
//	pdf := gofpdf.New("L", "mm", "A4", "")
//
//	// Добавляем шрифты
//	pdf.AddUTF8Font("montserrat", "", "Montserrat-Regular.ttf")
//	pdf.AddUTF8Font("montserrat", "B", "Montserrat-Bold.ttf")
//	pdf.AddUTF8Font("montserrat", "I", "Montserrat-Italic.ttf")
//
//	// 1. Титульная страница
//	createCoverPage(pdf, manager, phone)
//
//	// 2. Содержание (пока не знаем номер страницы таблицы)
//	// Сохраняем текущую страницу, чтобы потом вернуться
//	currentPage := pdf.PageNo()
//
//	// 3. Таблица моделей
//	tablePages := createModelTable(pdf, pictures)
//
//	// 4. Возвращаемся к содержанию и обновляем номера страниц
//	pdf.SetPage(currentPage)                  // Переходим на страницу содержания
//	createTableOfContents(pdf, currentPage+1) // Таблица начинается со следующей страницы
//
//	// 5. Заключительная страница
//	createConclusionPage(pdf)
//
//	// Сохраняем PDF
//	return pdf.OutputFileAndClose(outputPath)
//}
//
///**
//// Страница 1: Заставка
//func createCoverPage(pdf *gofpdf.Fpdf, manager, phone string) {
//	pdf.AddPage()
//
//	// Создаем цветной фон
//	pdf.SetFillColor(41, 128, 185) // Синий цвет
//	pdf.Rect(0, 0, 297, 210, "F")
//
//	// Заголовок
//	pdf.SetY(100)
//	pdf.SetFont("montserrat", "B", 24)
//	pdf.SetTextColor(255, 255, 255)
//	pdf.CellFormat(0, 20, "ОТЧЕТ ПО МОДЕЛЯМ", "", 0, "C", false, 0, "")
//
//	// Информация о менеджере
//	pdf.SetY(150)
//	pdf.SetFont("montserrat", "B", 16)
//	pdf.SetTextColor(255, 255, 255)
//	pdf.CellFormat(0, 10, "Почта: "+manager, "", 0, "C", false, 0, "")
//
//	// Номер телефона
//	pdf.SetY(170)
//	pdf.CellFormat(0, 10, "Телефон: "+phone, "", 0, "C", false, 0, "")
//
//	// Дата
//	pdf.SetY(250)
//	pdf.SetFont("montserrat", "I", 12)
//	pdf.CellFormat(0, 10, "Дата создания отчета", "", 0, "C", false, 0, "")
//}
//
//// Страница 2: Содержание
//func createTableOfContents(pdf *gofpdf.Fpdf) {
//	pdf.AddPage()
//
//	// Заголовок
//	pdf.SetFont("montserrat", "B", 20)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 20, "Содержание", "", 1, "C", false, 0, "")
//	pdf.Ln(10)
//
//	// Элементы содержания
//	pdf.SetFont("montserrat", "", 14)
//
//	// Таблица моделей
//	pdf.SetX(20)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 10, "Таблица моделей", "", 0, "L", false, 0, "")
//	pdf.SetX(170)
//	pdf.CellFormat(20, 10, "3", "", 1, "R", false, 0, "")
//
//	// Заключение
//	pdf.SetX(20)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 10, "Заключение", "", 0, "L", false, 0, "")
//	pdf.SetX(170)
//	pdf.CellFormat(20, 10, "4", "", 1, "R", false, 0, "")
//
//	pdf.Ln(20)
//
//	// Разделительная линия
//	pdf.SetDrawColor(200, 200, 200)
//	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
//}
//
//func createModelTable(pdf *gofpdf.Fpdf, pictures []*createpdffile.Models) {
//	pdf.SetFont("montserrat", "", 11)
//	rowHeight := 40.0
//	imageSize := 30.0
//
//	isFirstPage := true
//	modelsOnCurrentPage := 0
//	maxModelsFirstPage := 3
//	maxModelsOtherPages := 4
//
//	// ИЛИ вручную задать ширину (рекомендуется для контроля)
//	widths := []float64{
//		20,  // № (увеличили)
//		110, // Наименование (увеличили)
//		30,  // Кол-во
//		35,  // Цена
//		35,  // Сумма
//		30,  // Наличие
//		25,  // Фото
//	}
//	// Итого: 285мм (почти вся ширина с полями 10мм)
//
//	createHeaders := func() {
//		// Альбомная ориентация: 297мм ширина, минус поля (например, по 10мм с каждой стороны)
//		pageWidth := 297.0
//		leftMargin := 10.0
//		rightMargin := 10.0
//		usableWidth := pageWidth - leftMargin - rightMargin // 277мм доступно
//
//		headerHeight := 12.0
//
//		// Устанавливаем поля
//		pdf.SetLeftMargin(leftMargin)
//		pdf.SetRightMargin(rightMargin)
//
//		pdf.SetFillColor(200, 200, 200)
//		pdf.SetFont("montserrat", "B", 10)
//
//		// Распределяем ширину пропорционально
//		// Сумма ширин: 15+70+20+25+25+20+15 = 190 (из вашего примера)
//		// Нормализуем до usableWidth
//
//		// Соотношения (ширины из вашего примера)
//		originalWidths := []float64{15, 70, 20, 25, 25, 20, 15}
//		originalTotal := 190.0
//
//		// Масштабируем под новую ширину
//		scaleFactor := usableWidth / originalTotal
//		scaledWidths := make([]float64, len(originalWidths))
//
//		for i, w := range originalWidths {
//			scaledWidths[i] = w * scaleFactor
//		}
//
//		// Заголовки
//		headers := []string{
//			"№",
//			"Наименование, описание оборудования",
//			"Кол-во, шт.",
//			"Цена, руб.",
//			"Сумма, руб.",
//			"Наличие",
//			"Фото",
//		}
//
//		// Рисуем ячейки
//		for i, header := range headers {
//			if i == len(headers)-1 {
//				// Последняя ячейка - перенос строки
//				pdf.CellFormat(widths[i], headerHeight, header, "1", 1, "C", true, 0, "")
//			} else {
//				pdf.CellFormat(widths[i], headerHeight, header, "1", 0, "C", true, 0, "")
//			}
//		}
//
//		pdf.SetFont("montserrat", "", 11)
//		pdf.Ln(2)
//	}
//	pageNumber := 1
//
//	pdf.SetLeftMargin(10)
//	pdf.SetTopMargin(15)
//
//	for i, picture := range pictures {
//		// Если это начало новой страницы
//		if modelsOnCurrentPage == 0 {
//			// Добавляем новую страницу, если это не первая итерация
//			if i > 0 {
//				pdf.AddPage()
//				pageNumber++
//			}
//
//			// На первой странице добавляем заголовок
//			if isFirstPage && pageNumber == 1 {
//				createHeaders()
//			}
//			// На последующих страницах заголовка нет
//		}
//
//		// Определяем лимит для текущей страницы
//		maxModels := maxModelsOtherPages
//		if isFirstPage && pageNumber == 1 {
//			maxModels = maxModelsFirstPage
//		}
//
//		// Если достигли лимита на текущей странице - добавляем новую
//		if modelsOnCurrentPage >= maxModels {
//			pdf.AddPage()
//			pageNumber++
//			modelsOnCurrentPage = 0
//
//			// После первой страницы сбрасываем флаг
//			if isFirstPage {
//				isFirstPage = false
//			}
//
//			// На новых страницах заголовок не добавляем
//		}
//
//		// Чередование цвета фона
//		if modelsOnCurrentPage%2 == 0 {
//			pdf.SetFillColor(255, 255, 255) // Белый
//		} else {
//			pdf.SetFillColor(250, 250, 250) // Светло-серый
//		}
//
//		currentY := pdf.GetY()
//
//		// № строки
//		pdf.CellFormat(widths[0], rowHeight, strconv.Itoa(i+1), "1", 0, "C", true, 0, "")
//
//		// Название модели (обрезаем если слишком длинное)
//		name := picture.GetName()
//		maxNameLength := 50
//		if len(name) > maxNameLength {
//			name = name[:maxNameLength] + "..."
//		}
//		pdf.CellFormat(widths[1], rowHeight, name, "1", 0, "L", true, 0, "")
//
//		// Кол-во
//		pdf.CellFormat(widths[2], rowHeight, strconv.Itoa(int(picture.Count)), "1", 0, "C", true, 0, "")
//
//		// Цена
//		pdf.CellFormat(widths[3], rowHeight, "0", "1", 0, "C", true, 0, "")
//
//		// Сумма
//		pdf.CellFormat(widths[4], rowHeight, "0", "1", 0, "C", true, 0, "")
//
//		// Наличие
//		pdf.CellFormat(widths[5], rowHeight, "0", "1", 0, "C", true, 0, "")
//
//		// Ячейка для фото
//		xPos := pdf.GetX()
//		pdf.CellFormat(widths[6], rowHeight, "", "1", 1, "C", true, 0, "")
//
//		// Добавление изображения
//		if imagePath := picture.GetImg(); imagePath != "" {
//			imageX := xPos + (widths[6]-imageSize)/2
//			imageY := currentY + (rowHeight-imageSize)/2
//
//			if _, err := os.Stat(imagePath); err == nil {
//				// Пытаемся добавить изображение
//				pdf.Image(imagePath, imageX, imageY, imageSize, imageSize, false, "", 0, "")
//			} else {
//				// Если изображение не найдено - рисуем рамку
//				pdf.SetDrawColor(200, 200, 200)                      // Серый цвет рамки
//				pdf.SetFillColor(240, 240, 240)                      // Светло-серый фон
//				pdf.Rect(imageX, imageY, imageSize, imageSize, "FD") // F-заливка, D-рамка
//
//				// Текст "No Image"
//				pdf.SetTextColor(150, 150, 150) // Серый текст
//				pdf.SetFont("montserrat", "I", 6)
//				textX := imageX + (imageSize-pdf.GetStringWidth("No Image"))/2
//				textY := imageY + imageSize/2 + 2
//				pdf.Text(textX, textY, "No Image")
//				pdf.SetFont("montserrat", "", 11)
//				pdf.SetTextColor(0, 0, 0)
//			}
//		}
//
//		modelsOnCurrentPage++
//
//		if isFirstPage && pageNumber == 1 && modelsOnCurrentPage == maxModelsFirstPage {
//			// Не сбрасываем здесь, сбросим при добавлении новой страницы
//		}
//	}
//	//for i, picture := range pictures {
//	//	if modelsOnCurrentPage == 0 {
//	//		pdf.AddPage()
//	//		if isFirstPage {
//	//			createHeaders()
//	//		}
//	//	}
//	//
//	//	// Проверяем лимит для текущей страницы
//	//	maxModels := maxModelsOtherPages
//	//	if isFirstPage {
//	//		maxModels = maxModelsFirstPage
//	//	}
//	//
//	//	if modelsOnCurrentPage == maxModels {
//	//		pdf.AddPage()
//	//		modelsOnCurrentPage = 0
//	//		if isFirstPage {
//	//			isFirstPage = false
//	//		}
//	//	}
//	//
//	//	// Чередование цвета фона
//	//	if modelsOnCurrentPage%2 == 0 {
//	//		pdf.SetFillColor(255, 255, 255)
//	//	} else {
//	//		pdf.SetFillColor(250, 250, 250)
//	//	}
//	//
//	//	currentY := pdf.GetY()
//	//
//	//	// № строки (ширина увеличена)
//	//	pdf.CellFormat(15, rowHeight, strconv.Itoa(i+1), "1", 0, "C", true, 0, "")
//	//
//	//	// Название модели (ширина увеличена)
//	//	pdf.CellFormat(70, rowHeight, picture.GetName(), "1", 0, "C", true, 0, "")
//	//
//	//	// Кол-во
//	//	pdf.CellFormat(20, rowHeight, strconv.Itoa(int(picture.Count)), "1", 0, "C", true, 0, "")
//	//
//	//	// Цена (заглушка)
//	//	pdf.CellFormat(25, rowHeight, "0", "1", 0, "C", true, 0, "")
//	//
//	//	// Сумма (заглушка)
//	//	pdf.CellFormat(25, rowHeight, "0", "1", 0, "C", true, 0, "")
//	//
//	//	// Наличие (ширина уменьшена)
//	//	pdf.CellFormat(20, rowHeight, "0", "1", 0, "C", true, 0, "")
//	//
//	//	// Ячейка для фото (ширина уменьшена)
//	//	xPos := pdf.GetX()
//	//	pdf.CellFormat(15, rowHeight, "", "1", 1, "C", true, 0, "")
//	//
//	//	if imagePath := picture.GetImg(); imagePath != "" {
//	//		imageX := xPos + (15-imageSize)/2
//	//		imageY := currentY + (rowHeight-imageSize)/2
//	//
//	//		if _, err := os.Stat(imagePath); err == nil {
//	//			pdf.Image(imagePath, imageX, imageY, imageSize, imageSize, false, "", 0, "")
//	//		} else {
//	//			pdf.SetDrawColor(200, 200, 200)                      // Серый цвет рамки
//	//			pdf.SetFillColor(240, 240, 240)                      // Светло-серый фон
//	//			pdf.Rect(imageX, imageY, imageSize, imageSize, "FD") // F-заливка, D-рамка
//	//
//	//			pdf.SetTextColor(150, 150, 150) // Серый текст
//	//			pdf.SetFont("Arial", "I", 6)    // Еще меньший размер шрифта
//	//			textX := imageX + (imageSize-pdf.GetStringWidth("No Image"))/2
//	//			textY := imageY + imageSize/2 + 2
//	//			pdf.Text(textX, textY, "No Image")
//	//
//	//			pdf.SetTextColor(0, 0, 0)
//	//			pdf.SetFont("montserrat", "", 11)
//	//		}
//	//	}
//	//
//	//	modelsOnCurrentPage++
//	//
//	//	// После добавления 3-й модели на первой странице переключаемся
//	//	if isFirstPage && modelsOnCurrentPage == maxModelsFirstPage {
//	//		isFirstPage = false
//	//	}
//	//}
//}
//
//// Страница 4: Заключительная страница
//func createThankYouPage(pdf *gofpdf.Fpdf) {
//	pdf.AddPage()
//
//	// Заголовок
//	pdf.SetFont("montserrat", "B", 24)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 100, "Спасибо за внимание!", "", 1, "C", false, 0, "")
//
//	// Дополнительная информация
//	pdf.SetFont("montserrat", "", 14)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.CellFormat(0, 20, "Отчет был сгенерирован автоматически.", "", 1, "C", false, 0, "")
//	pdf.CellFormat(0, 20, "Благодарим за использование нашего сервиса.", "", 1, "C", false, 0, "")
//}
//
//*/
