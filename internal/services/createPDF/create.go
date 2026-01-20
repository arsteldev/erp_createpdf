package createPDF

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	createpdffile "github.com/arsteldev/createPDF/proto"
	imaging "github.com/disintegration/imaging"
	"github.com/jung-kurt/gofpdf"
	"image"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type PDFServer struct {
	createpdffile.UnimplementedPDFCreatorServer
	Log *slog.Logger
}

type LinkItem struct {
	ShortName string
	Name      string
	ID        int
	Page      int
	Main      bool
}

// Константы для таблицы моделей
const (
	rowHeight           = 36.686
	imageSize           = 25.0
	maxModelsFirstPage  = 3
	maxModelsOtherPages = 4
	math                = (25.4 / 72.0)
)

type PowerData struct {
	Label string
	Value interface{}
}

// Ширины для альбомной ориентации
var tableWidths = []float64{
	10,  // № (шире для двузначных номеров)
	100, // Наименование - ОЧЕНЬ широкая в горизонтали!
	11,  // Кол-во шт.
	27,  // Цена
	27,  // Сумма
	16,  // Наличие
	41,  // Фото (можно сделать побольше для фото)
}

var site string

// Вспомогательная функция для цвета строки
//type RGBColor struct {
//	R, G, B int
//}

func (s *PDFServer) CreatePDF(ctx context.Context, req *createpdffile.CreatePDFRequest) (*createpdffile.CreatePDFResponse, error) {
	s.Log.Info("Received request from email",
		slog.String("email", req.GetFirstpage().GetContacts().GetEmail()),
		slog.String("phone", req.GetFirstpage().GetContacts().GetPhone()),
		slog.Int("models_count", len(req.GetModels())))

	pdfBytes, err := s.generatePDF(req)
	if err != nil {
		s.Log.Error("Ошибка создания PDF", slog.String("error", err.Error()))
		return &createpdffile.CreatePDFResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	s.Log.Info("Успешное создание PDF файла", slog.Int("bytes", len(pdfBytes)))
	return &createpdffile.CreatePDFResponse{
		Pdf:     pdfBytes,
		Success: true,
	}, nil
}

func (s *PDFServer) generatePDF(req *createpdffile.CreatePDFRequest) ([]byte, error) {
	fontDir := "/var/www/createpdf/font/"

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
		FontDirStr:     fontDir,
	})

	// Отключаем авторазрыв страницы для полного контроля
	pdf.SetAutoPageBreak(false, 10)

	fontFiles := map[string]struct {
		filepath string
		family   string
		style    string
	}{
		"montserrat-regular":  {"Montserrat-Regular.ttf", "montserrat", ""},
		"montserrat-bold":     {"Montserrat-Bold.ttf", "montserrat", "B"},
		"montserrat-italic":   {"Montserrat-Italic.ttf", "montserrat", "I"},
		"montserrat-medium":   {"Montserrat-Medium.ttf", "inter", "M"},
		"montserrat-semibold": {"Montserrat-SemiBold.ttf", "inter", "SB"},
		"inter-regular":       {"Inter-Regular.ttf", "inter", ""},
		"inter-bold":          {"Inter-Bold.ttf", "inter", "B"},
		"inter-italic":        {"Inter-Italic.ttf", "inter", "I"},
		"inter-medium":        {"Inter-Medium.ttf", "inter", "M"},
		"inter-semibold":      {"Inter-SemiBold.ttf", "inter", "SB"},
	}

	// Проверяем все шрифты
	allFontsAvailable := true
	for name, font := range fontFiles {
		fullPath := fontDir + font.filepath
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			s.Log.Error("Файл шрифта не найден",
				slog.String("name", name),
				slog.String("path", fullPath))
			allFontsAvailable = false
		} else {
			s.Log.Debug("Шрифт доступен",
				slog.String("name", name),
				slog.String("file", font.filepath))
		}
	}

	// Если какие-то шрифты отсутствуют, используем DejaVu как fallback
	if !allFontsAvailable {
		s.Log.Warn("Некоторые шрифты не найдены, используем DejaVu как запасной вариант")

		// Добавляем DejaVu шрифты
		dejaVuAdded := true
		dejaVuFonts := map[string]string{
			"regular": "DejaVuSans.ttf",
			"bold":    "DejaVuSans-Bold.ttf",
			"italic":  "DejaVuSans-Oblique.ttf",
		}

		for style, filename := range dejaVuFonts {
			fullPath := fontDir + filename
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				s.Log.Error("Файл DejaVu шрифта не найден",
					slog.String("style", style),
					slog.String("file", filename))
				dejaVuAdded = false
			}
		}

		if dejaVuAdded {
			pdf.AddUTF8Font("dejavu", "", "DejaVuSans.ttf")
			pdf.AddUTF8Font("dejavu", "B", "DejaVuSans-Bold.ttf")
			pdf.AddUTF8Font("dejavu", "I", "DejaVuSans-Oblique.ttf")
			pdf.SetFont("dejavu", "", 12)
		} else {
			s.Log.Error("DejaVu шрифты также не найдены, используем стандартные шрифты")
		}
	} else {
		// Добавляем все шрифты, если они все есть
		pdf.AddUTF8Font("montserrat", "", "Montserrat-Regular.ttf")
		pdf.AddUTF8Font("montserrat", "B", "Montserrat-Bold.ttf")
		pdf.AddUTF8Font("montserrat", "I", "Montserrat-Italic.ttf")
		pdf.AddUTF8Font("montserrat", "M", "Montserrat-Medium.ttf")
		pdf.AddUTF8Font("montserrat", "SB", "Montserrat-SemiBold.ttf")

		pdf.AddUTF8Font("inter", "", "Inter-Regular.ttf")
		pdf.AddUTF8Font("inter", "B", "Inter-Bold.ttf")
		pdf.AddUTF8Font("inter", "I", "Inter-Italic.ttf")
		pdf.AddUTF8Font("inter", "M", "Inter-Medium.ttf")
		pdf.AddUTF8Font("inter", "SB", "Inter-SemiBold.ttf")

		// Устанавливаем шрифт по умолчанию
		pdf.SetFont("montserrat", "", 14)
	}

	// Устанавливаем автора и заголовок
	pdf.SetAuthor(req.GetFirstpage().GetContacts().GetEmail(), true)
	pdf.SetTitle("Отчет по моделям", true)

	site = req.GetFirstpage().GetContacts().GetSite()
	pdf.SetLeftMargin(32)
	pdf.SetRightMargin(32)

	// 1. Титульная страница
	createCoverPage(pdf, req.GetFirstpage())

	// 2. Создаем ссылки ДО создания содержания
	links := make(map[string]LinkItem)
	order := make([]string, 0)
	addLink := func(key string, item LinkItem) {
		order = append(order, key)
		links[key] = item
	}

	// Добавляем один элемент
	if len(req.GetFeatures()) != 0 {
		addLink("features", LinkItem{
			ShortName: "features",
			Name:      "Особенности системы и требования заказчика",
			ID:        pdf.AddLink(),
			Page:      0,
			Main:      true,
		})
	}

	if req.GetSelectequipment().GetTextEquipment() != "" && len(req.GetSelectequipment().GetSchema()) > 0 && len(req.GetSelectequipment().GetAccommodation()) > 0 {
		addLink("selectsEquipments", LinkItem{
			ShortName: "selectsEquipments",
			Name:      "Выбор оборудования",
			ID:        pdf.AddLink(),
			Page:      0,
			Main:      true,
		})

		addLink("schemaEquipments", LinkItem{
			ShortName: "schemaEquipments",
			Name:      "Структурная схема проекта",
			ID:        pdf.AddLink(),
			Page:      0,
			Main:      false,
		})

		addLink("accommodationEquipments", LinkItem{
			ShortName: "accommodationEquipments",
			Name:      "Размещение блоков системы в шкафах",
			ID:        pdf.AddLink(),
			Page:      0,
			Main:      false,
		})
	}

	// Добавляем группу элементов (исправлен синтаксис []LinkItem{...}...)
	addLink("tabelModel", LinkItem{
		ShortName: "tabelModel",
		Name:      "Спецификация оборудования",
		ID:        pdf.AddLink(),
		Page:      0,
		Main:      true,
	})

	addLink("characteristics", LinkItem{
		ShortName: "characteristics",
		Name:      "Характеристики системы",
		ID:        pdf.AddLink(),
		Page:      0,
		Main:      false,
	})

	//specLink := pdf.AddLink() // Ссылка на спецификацию оборудования
	//charLink := pdf.AddLink() // Ссылка на характеристики системы

	// 3. Добавляем страницу содержания (пока без номеров)
	pdf.AddPage()
	// Передаем 0 как номера страниц, но уже передаем ID ссылок
	createTableOfContents(pdf, 0, 0, links, order)

	// 4. Запоминаем текущую страницу (содержание)
	tocPageNum := pdf.PageNo()

	number := 0

	if len(req.GetFeatures()) != 0 {
		pdf.AddPage()
		tempLink := links["features"]
		tempLink.Page = pdf.PageNo()
		links["features"] = tempLink

		pdf.SetLink(links["features"].ID, 0, links["features"].Page)
		number++
		createSystemFeatures(pdf, req.GetFeatures(), number)
	}

	if req.GetSelectequipment().GetTextEquipment() != "" && len(req.GetSelectequipment().GetSchema()) > 0 && len(req.GetSelectequipment().GetAccommodation()) > 0 {
		pdf.AddPage()
		tempLink := links["selectsEquipments"]
		tempLink.Page = pdf.PageNo()
		links["selectsEquipments"] = tempLink

		pdf.SetLink(links["selectsEquipments"].ID, 0, links["selectsEquipments"].Page)
		number++
		selectsEquipments(pdf, req.GetSelectequipment().GetTextEquipment(), number)

		pdf.AddPage()
		tempLink = links["schemaEquipments"]
		tempLink.Page = pdf.PageNo()
		links["schemaEquipments"] = tempLink

		pdf.SetLink(links["schemaEquipments"].ID, 0, links["schemaEquipments"].Page)
		imageEquipments(pdf, req.GetSelectequipment().GetSchema(), links["schemaEquipments"].Name, "schema", number, 1)

		/**
		addLink("accommodationEquipments", LinkItem{
			ShortName: "accommodationEquipments",
			Name:      "Размещение блоков системы в шкафах",
			ID:        pdf.AddLink(),
			Page:      0,
			Main:      false,
		})
		*/
		pdf.AddPage()
		tempLink = links["accommodationEquipments"]
		tempLink.Page = pdf.PageNo()
		links["accommodationEquipments"] = tempLink

		pdf.SetLink(links["accommodationEquipments"].ID, 0, links["accommodationEquipments"].Page)
		imageEquipments(pdf, req.GetSelectequipment().GetAccommodation(), links["accommodationEquipments"].Name, "accommodation", number, 2)

	}

	// 5. Создаем таблицу моделей (Спецификация оборудования)
	pdf.AddPage()
	tempLink := links["tabelModel"]
	tempLink.Page = pdf.PageNo()
	links["tabelModel"] = tempLink

	// Устанавливаем место назначения для ссылки на спецификацию
	// Нужно установить ссылку до вызова createModelTable
	pdf.SetLink(links["tabelModel"].ID, 0, links["tabelModel"].Page)

	// Создаем таблицу
	number++
	_ = createModelTable(pdf, req.GetModels(), req.GetAmount(), number)

	// 6. Создаем Таблицу характеристик
	pdf.AddPage()
	//characteristicsStartPage := pdf.PageNo()
	tempLink = links["characteristics"]
	tempLink.Page = pdf.PageNo()
	links["characteristics"] = tempLink

	// Устанавливаем место назначения для ссылки на характеристики
	pdf.SetLink(links["characteristics"].ID, 0, links["characteristics"].Page)

	// Создаем характеристики
	_ = createCharacteristic(pdf, req.GetModels(), req.GetModelsdata(), number)

	// 7. ВОЗВРАЩАЕМСЯ на страницу содержания для обновления номеров
	currentPageBeforeUpdate := pdf.PageNo() // Сохраняем текущую страницу
	pdf.SetPage(tocPageNum)                 // Переходим на страницу содержания

	// Очищаем страницу и перерисовываем содержание с правильными номерами
	// Важно: сначала очистить область
	pageWidth, pageHeight := pdf.GetPageSize()
	pdf.SetFillColor(255, 255, 255)            // Белый цвет
	pdf.Rect(0, 0, pageWidth, pageHeight, "F") // Закрашиваем всю страницу

	// Устанавливаем позицию и отступы
	pdf.SetY(44) // Начальная позиция как в createTableOfContents
	pdf.SetLeftMargin(78)
	pdf.SetRightMargin(32)

	// Перерисовываем содержание с актуальными номерами страниц и теми же ссылками
	createTableOfContents(pdf, links["tabelModel"].Page, links["characteristics"].Page, links, order)

	// Добавляем вотермарку на страницу содержания
	addWatermark(pdf)

	// 8. ВАЖНО: Возвращаемся на последнюю страницу
	pdf.SetPage(currentPageBeforeUpdate)

	//pdf.SetPage(currentPageBeforeUpdate)

	//s.Log.Warn("Features fetched", "features", req.GetFeatures())

	// Сохраняем в буфер
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания буфера PDF: %v", err)
	}

	// Дополнительно сохраняем в файл (для отладки)
	exePath, err := os.Executable()
	if err != nil {
		s.Log.Error("Ошибка получения пути исполняемого файла", slog.String("error", err.Error()))
	} else {
		outputDir := filepath.Dir(exePath)
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("report_%s.pdf", timestamp)
		fullPath := filepath.Join(outputDir, filename)

		err = os.WriteFile(fullPath, buf.Bytes(), 0644)
		if err != nil {
			s.Log.Error("Ошибка сохранения в файл", slog.String("error", err.Error()))
		} else {
			s.Log.Info("PDF файл успешно сохранен", slog.String("path", fullPath))
		}
	}

	return buf.Bytes(), nil
}

// Страница 1: Заставка
// func createCoverPage(pdf *gofpdf.Fpdf, manager, phone string, startKpImage, companyBigImage, smallCoCompanyImage []byte) {
func createCoverPage(pdf *gofpdf.Fpdf, firstPage *createpdffile.FirstPage) {
	pdf.AddPage()

	// 1. Фоновая картинка (startKpImage) - на всю страницу
	pageWidth, pageHeight := pdf.GetPageSize()
	setImageIntoPDF(pdf, firstPage.GetStartKpImage(), 0, 0, pageWidth, pageHeight, "coverBackground", true)

	// 2. Логотип компании в левом углу (companyBigImage)
	setImageIntoPDF(pdf, firstPage.GetCompanyBigImage(), 34, 17, 60, 15, "companyLogo", false)

	// 3. Малый логотип в правом углу (smallCoCompanyImage)
	setImageIntoPDF(pdf, firstPage.GetSmallCoCompanyImage(), pageWidth-64, 19, 32, 9, "smallLogo", false)

	// 4. Картинка, которая стоит в по центру 1 страницы
	setImageIntoPDF(pdf, firstPage.GetMainPageImage(), pageWidth-120, pageHeight-130, 110, 50, "mainPageLogo", false)

	// Картинки, которые я буду использовать в addWaterMark легче было загрузить сразу, а потом только использовать по их имени.
	// Именно поэтому они и 1 и 1 и -10, -10
	setImageIntoPDF(pdf, firstPage.GetSmallCoCompanyImage(), -10, -10, 1, 1, "leftImageIntoWaterMark", false)
	setImageIntoPDF(pdf, firstPage.GetCompanySmall(), -10, -10, 1, 1, "rightImageIntoWaterMark", false)

	// Заголовок
	pdf.SetFont("montserrat", "M", 34)
	pdf.SetTextColor(37, 36, 36)
	pdf.SetY(47)
	pdf.MultiCell(0, 38*math, "КОММЕРЧЕСКОЕ\nПРЕДЛОЖЕНИЕ", "", "L", false)
	//pdf.CellFormat(0, 20, "", "", 1, "L", false, 0, "")
	//pdf.SetXY(30, 55)
	//pdf.CellFormat(0, 20, "ПРЕДЛОЖЕНИЕ", "", 1, "L", false, 0, "")

	// Исх № От Дата
	textStr := fmt.Sprintf("Исх. № %d от %s г.", firstPage.GetId(), time.Now().Format("02.01.2006"))
	pdf.SetFont("montserrat", "M", 9)
	pdf.SetTextColor(254, 80, 0)
	pdf.SetY(87)
	pdf.MultiCell(0, 10.8*math, textStr, "", "L", false)

	// Информация о человеке которому скидываем ПДФ
	pdf.SetFont("inter", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(104)
	pdf.MultiCell(0, 14.4*math, firstPage.GetContactuser().GetFullName(), "", "L", false)

	pdf.SetFont("inter", "I", 10)
	pdf.SetTextColor(0, 0, 0)
	drawContactInfo(pdf, firstPage.GetContactuser())

	// Объект
	pdf.SetFont("inter", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(122)
	pdf.CellFormat(0, 14.4*math, "Объект:", "", 1, "L", false, 0, "")

	pdf.SetFont("inter", "", 12)
	pdf.SetY(126)
	pdf.CellFormat(0, 14.4*math, firstPage.GetObject(), "", 1, "L", false, 0, "")

	// Кто выполнил (Менеджер)
	pdf.SetFont("inter", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(138)
	pdf.CellFormat(0, 14.4*math, "Выполнил:", "", 1, "L", false, 0, "")
	//pdf.SetY(138)
	pdf.SetFont("inter", "", 12)
	pdf.CellFormat(0, 14.4*math, firstPage.GetWhocreate().GetFullName(), "", 1, "L", false, 0, "")
	pdf.SetFont("inter", "I", 10)
	//pdf.SetY(143)
	pdf.CellFormat(0, 12*math, firstPage.GetWhocreate().GetOccupy(), "", 1, "L", false, 0, "")

	// Контактная информация
	pdf.SetFont("inter", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(153)
	pdf.CellFormat(0, 14.4*math, "Контакты:", "", 1, "L", false, 0, "")

	pdf.SetFont("inter", "", 12)
	pdf.SetY(158)
	contactText := "тел.:" + firstPage.GetContacts().GetPhone() + "\n" + "e-mail.:" + firstPage.GetContacts().GetEmail() + "\n" + site
	pdf.MultiCell(0, 14.4*math, contactText, "", "L", false)
}

func drawContactInfo(pdf *gofpdf.Fpdf, contactUser *createpdffile.ContactUser) {
	occupy := contactUser.GetOccupy()
	fsOrg := contactUser.GetFsOrg()
	nameOrg := contactUser.GetNameOrg()

	fullText := strings.TrimSpace(fmt.Sprintf("%s %s %s", occupy, fsOrg, nameOrg))

	if len(fullText) <= 45 {
		pdf.MultiCell(0, 12*math, fullText, "", "L", false)
	} else {
		pdf.MultiCell(0, 12*math, occupy+"\n"+fmt.Sprintf("%s %s", fsOrg, nameOrg), "", "L", false)
	}
}

func convertPNG16to8(imageData []byte) ([]byte, error) {
	// Декодируем изображение
	img, err := png.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}

	// Конвертируем в 8-бит
	img8 := imaging.Clone(img)

	// Кодируем обратно в PNG-8
	var buf bytes.Buffer
	encoder := png.Encoder{
		CompressionLevel: png.DefaultCompression,
	}
	err = encoder.Encode(&buf, img8)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Страница 2: Содержание
func createTableOfContents(pdf *gofpdf.Fpdf, tableStartPage, characteristicsPage int, links map[string]LinkItem, order []string) {
	// Устанавливаем отступы
	pdf.SetLeftMargin(78)
	pdf.SetTopMargin(44)
	pdf.SetY(44)

	// Заголовок
	pdf.SetFont("montserrat", "", 24)
	pdf.SetTextColor(37, 36, 36)
	pdf.CellFormat(0, 24*math, "СОДЕРЖАНИЕ", "", 0, "L", false, 0, "")
	pdf.Ln(40 * math)

	// Шрифт для содержания
	pdf.SetFont("inter", "", 12)

	// Функция для добавления пункта содержания
	addTOCItem := func(text string, pageNum int, link int, main bool) {
		y := pdf.GetY()

		// Смещение и оформление текста
		startX := 78
		if main {
			pdf.SetTextColor(237, 114, 3)
			text = strings.ToUpper(text)
		} else {
			pdf.SetTextColor(0, 0, 0)
			text = "     " + text // 5 пробелов
			startX += 5           // визуальное смещение
		}

		pdf.SetX(float64(startX))

		// Рисуем основной текст
		if link > 0 {
			pdf.CellFormat(0, 16*math, text, "", 0, "L", false, link, "")
		} else {
			pdf.CellFormat(0, 16*math, text, "", 0, "L", false, 0, "")
		}

		// Ширины
		textWidth := pdf.GetStringWidth(text)
		pageWidth, _ := pdf.GetPageSize()

		dotStartX := float64(startX) + textWidth + 5

		// Номер страницы
		if pageNum > 0 {
			var pageNumStr string
			if pageNum <= 9 {
				pageNumStr = "0" + strconv.Itoa(pageNum)
			} else {
				pageNumStr = strconv.Itoa(pageNum)
			}
			pageNumWidth := pdf.GetStringWidth(pageNumStr)

			// было: pageNumX := pageWidth - 32 - 65
			pageNumX := pageWidth - 32 - 65 + 10 // сдвиг номера на 10mm вправо

			// было: availableWidth := pageNumX - float64(startX) - 1
			availableWidth := pageNumX + 3 - dotStartX - 1 // ширина точек ДО номера

			if availableWidth > 0 {
				dotWidth := pdf.GetStringWidth(".")
				numDots := int(availableWidth / dotWidth)
				if numDots <= 0 {
					numDots = 1
				}

				dotText := strings.Repeat(".", numDots)
				pdf.SetXY(dotStartX, y)
				pdf.SetTextColor(120, 120, 120)
				pdf.CellFormat(availableWidth, 16*math, dotText, "", 0, "L", false, 0, "")

				// Номер страницы
				pdf.SetXY(pageNumX-pageNumWidth, y)
				pdf.SetTextColor(0, 0, 0)

				if link > 0 {
					pdf.CellFormat(pageNumWidth+10, 16*math, pageNumStr, "", 0, "R", false, link, "")
				} else {
					pdf.CellFormat(pageNumWidth+10, 16*math, pageNumStr, "", 0, "R", false, 0, "")
				}
			}

		}

		pdf.Ln(16 * math)
	}

	// Добавляем пункты с ссылками
	for _, key := range order {
		item := links[key]
		addTOCItem(item.Name, item.Page, item.ID, item.Main)
	}

	//addTOCItem("Спецификация оборудования", tableStartPage, specLinkID)
	//addTOCItem("Характеристики системы", characteristicsPage, charLinkID)

	if links["characteristics"].Page == 0 {
		addWatermark(pdf)
	}
}

func createModelTable(pdf *gofpdf.Fpdf, pictures []*createpdffile.Models, amount *createpdffile.Amount, number int) int {
	modelsOnCurrentPage := 0
	tablePageNumber := 1
	totalModels := len(pictures)
	icon := pictures[0].Icon
	needRub := false
	var needText string
	switch icon {
	case "₽":
		needRub = true
		break
	}

	// Настройка PDF
	pdf.SetLeftMargin(32)
	pdf.SetRightMargin(32)
	pdf.SetTopMargin(15)

	// ПЕРВАЯ СТРАНИЦА
	setSpecificationEquipment(pdf, 2, number)
	pdf.SetY(50)
	addWatermark(pdf)
	drawTableHeaderForLandscape(pdf, tableWidths, 16.932)

	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.3)

	// Обработка первой страницы (максимум 3 модели)
	firstPageLimit := 3
	modelsDrawn := 0

	for i := 0; i < totalModels && i < firstPageLimit; i++ {
		rowColor := getRowColor(i)
		pdf.SetFillColor(rowColor.R, rowColor.G, rowColor.B)
		drawTableRow(pdf, i, pictures[i], tableWidths, i%2 == 0)
		if pictures[i].GetPresence() == "Заказ" {
			needText = "По условиям договора поставка осуществляется при 100% предоплате со склада в Санкт-Петербурге.\nЦены указаны с учетом НДС 22%. Срок поставки оборудования под заказ  –  3 месяца с момента оплаты счета."
		}
		modelsDrawn++
		modelsOnCurrentPage++
	}

	// Если на первой странице есть место для итогов (моделей < 3)
	if totalModels < 3 {
		// Итоги на первой странице
		createSumm(pdf, amount, modelsDrawn%2 == 0, needRub, needText)
		return 1 // только одна страница
	}

	// Если ровно 3 модели - новая страница для итогов
	if totalModels == 3 {
		pdf.AddPage()
		tablePageNumber++
		// Только шапка на странице с итогами
		addWatermark(pdf)
		pdf.SetY(20)
		drawTableHeaderForLandscape(pdf, tableWidths, 14.11)
		createSumm(pdf, amount, true, needRub, needText)
		return 2
	}

	// Если больше 3 моделей
	modelsOnCurrentPage = 0

	// ВТОРАЯ И ПОСЛЕДУЮЩИЕ СТРАНИЦЫ
	for i := 3; i < totalModels; i++ {
		// Если это начало новой страницы
		if modelsOnCurrentPage == 0 {
			pdf.AddPage()
			tablePageNumber++
			pdf.SetY(17)
			drawTableHeaderForLandscape(pdf, tableWidths, 14.11)
			addWatermark(pdf)
		}

		rowColor := getRowColor(modelsOnCurrentPage)
		pdf.SetFillColor(rowColor.R, rowColor.G, rowColor.B)
		drawTableRow(pdf, i, pictures[i], tableWidths, modelsOnCurrentPage%2 == 0)
		if pictures[i].GetPresence() == "Заказ" {
			needText = "По условиям договора поставка осуществляется при 100% предоплате со склада в Санкт-Петербурге.\nЦены указаны с учетом НДС 20%. Срок поставки оборудования под заказ  –  3 месяца с момента оплаты счета."
		}
		modelsOnCurrentPage++
		modelsDrawn++

		// Проверяем лимит в 4 строки на странице
		if modelsOnCurrentPage >= 4 {
			// Если это последняя модель и страница полная - итоги на следующей
			if i == totalModels-1 {
				pdf.AddPage()
				tablePageNumber++
				pdf.SetY(20)
				drawTableHeaderForLandscape(pdf, tableWidths, 14.11)
				addWatermark(pdf)
			}
			modelsOnCurrentPage = 0
		}
	}

	createSumm(pdf, amount, modelsDrawn%2 == 1, needRub, needText)

	return tablePageNumber
}

func createSumm(pdf *gofpdf.Fpdf, amount *createpdffile.Amount, isEvenRow, needRub bool, needText string) int {
	var fillColor RGBColor
	if isEvenRow {
		fillColor = RGBColor{232, 237, 237}
	} else {
		fillColor = RGBColor{255, 255, 255}
	}

	// ПЕРВАЯ СТРОКА
	pdf.SetFillColor(fillColor.R, fillColor.G, fillColor.B)
	pdf.SetDrawColor(fillColor.R, fillColor.G, fillColor.B)
	pdf.SetLineWidth(0.1)
	pdf.SetTextColor(255, 89, 3)
	pdf.SetFont("Inter", "", 12)

	//var amountText string
	amountText := fmt.Sprintf("%.2f"+amount.GetIcon(), float64(amount.GetMoney())/100.0)

	//if needRub {
	//	amountText = fmt.Sprintf("%.2f₽", float64(amount.Rub)/100.0)
	//} else {
	//	amountText = fmt.Sprintf("%.2f $/%.2f €", float64(amount.Usd)/100.0, float64(amount.Eur)/100.0)
	//
	//}

	text := "ИТОГО:"
	spaces := strings.Repeat(" ", 113)
	spacesFirst := strings.Repeat(" ", 1)
	fullText := spacesFirst + text + spaces + amountText

	// Получаем текущую позицию
	currentY := pdf.GetY()

	// Получаем ширину страницы
	pageWidth, _ := pdf.GetPageSize()
	leftMargin := 32.0 * math
	rightMargin := 32.0 * math
	availableWidth := pageWidth - leftMargin - rightMargin

	// 1. Закрашиваем фон от левого до правого края
	pdf.SetXY(0, currentY)
	pdf.CellFormat(pageWidth, rowHeight/2, "", "0", 0, "C", true, 0, "")

	// 2. Возвращаемся на исходную позицию для текста
	pdf.SetY(currentY) // Или ваша исходная X позиция
	pdf.CellFormat(pageWidth, rowHeight/2, fullText, "1", 0, "LM", false, 0, "")

	if needText != "" {
		if !isEvenRow {
			fillColor = RGBColor{232, 237, 237}
		} else {
			fillColor = RGBColor{255, 255, 255}
		}
		pdf.SetFont("Inter", "", 12)

		// ПЕРВАЯ СТРОКА
		pdf.SetFillColor(fillColor.R, fillColor.G, fillColor.B)
		pdf.SetDrawColor(fillColor.R, fillColor.G, fillColor.B)
		pdf.SetLineWidth(0.1)
		// Переход на следующую строку
		pdf.Ln(rowHeight / 2)

		// Получаем новую позицию Y
		currentY = pdf.GetY()

		// 3. Закрашиваем фон для второй строки
		pdf.SetXY(0, currentY)
		pdf.CellFormat(availableWidth, rowHeight/2, "", "0", 0, "C", true, 0, "")

		// Возвращаемся на исходную позицию для текста
		pdf.SetXY(32, currentY)

		// ВТОРАЯ СТРОКА (с другим фоном)
		needTextWithIndent := strings.Replace(needText, "\n", "\n"+strings.Repeat(" ", 1), -1)
		needTextWithIndent = strings.Repeat(" ", 1) + needTextWithIndent
		pdf.SetFont("Inter", "", 10.5)

		lineHeight := rowHeight/2 - 10
		pdf.MultiCell(233, lineHeight, needTextWithIndent, "1", "L", false)

		// Сбросить отступ
		pdf.SetLeftMargin(0)
	}

	return 2
}

func getRowColor(rowIndex int) RGBColor {
	if rowIndex%2 == 0 {
		return RGBColor{232, 237, 237} // #e8eded
	}
	return RGBColor{255, 255, 255} // #ffffff
}

// Функция для рисования шапки таблицы для горизонтали
func drawTableHeaderForLandscape(pdf *gofpdf.Fpdf, widths []float64, headerHeights float64) {
	// Темно-серый фон #3d4a4d
	pdf.SetFillColor(61, 74, 77)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Inter", "B", 9.5)

	// Для шапки делаем границы того же цвета что и фон
	pdf.SetDrawColor(61, 74, 77)

	headers := []string{
		"№",
		"Наименование,\nописание оборудования",
		"Кол-во,\nшт.",
		"Цена",
		"Сумма",
		"Наличие",
		"Фото",
	}

	headerHeight := headerHeights
	currentY := pdf.GetY()
	startX := pdf.GetX()

	// 1. Сначала закрашиваем всю полосу от левого края до правого
	pageWidth, _ := pdf.GetPageSize()
	leftMargin := 0.0
	rightMargin := 0.0

	// Закрашиваем всю полосу
	pdf.SetXY(leftMargin, currentY)
	pdf.CellFormat(pageWidth-leftMargin-rightMargin, headerHeight, "", "0", 0, "C", true, 0, "")

	// 2. Теперь рисуем текст в ячейках
	// Функция для рисования ячейки с многострочным текстом
	drawHeaderCell := func(x, y, width, height float64, text, align string) {
		// Рассчитываем параметры для текста
		lines := strings.Split(text, "\n")
		lineCount := len(lines)

		// Межстрочный интервал - 2 мм (в точках)
		lineSpacing := 2.0 * math // 2 мм в точках
		textLineHeight := 5.0     // Высота одной строки текста

		// Общая высота текстового блока
		textBlockHeight := float64(lineCount)*textLineHeight + float64(lineCount-1)*lineSpacing

		// Начальная позиция Y для вертикального центрирования
		startTextY := y + (height-textBlockHeight)/2

		// Рисуем каждую строку
		for i, line := range lines {
			yPos := startTextY + float64(i)*(textLineHeight+lineSpacing)

			pdf.SetXY(x, yPos)
			pdf.CellFormat(width, textLineHeight, line, "", 0, align, false, 0, "")
		}
	}

	// Рисуем все ячейки заголовка
	x := startX
	for i, header := range headers {
		align := "C"
		if i == 1 {
			align = "L"
		}

		drawHeaderCell(x, currentY, widths[i], headerHeight, header, align)
		x += widths[i]
	}

	// Устанавливаем позицию для следующей строки
	pdf.SetXY(startX, currentY+headerHeight)
}

// Функция для рисования строки таблицы
func drawTableRow(pdf *gofpdf.Fpdf, index int, picture *createpdffile.Models, widths []float64, isEvenRow bool) {
	// Определяем цвет фона строки
	var fillColor RGBColor
	if isEvenRow {
		fillColor = RGBColor{232, 237, 237} // #e8eded - серая строка
	} else {
		fillColor = RGBColor{255, 255, 255} // #ffffff - белая строка
	}

	// Устанавливаем цвет заливки и границы
	pdf.SetFillColor(fillColor.R, fillColor.G, fillColor.B)
	pdf.SetDrawColor(fillColor.R, fillColor.G, fillColor.B) // Границы того же цвета!
	pdf.SetLineWidth(0.1)

	currentY := pdf.GetY()
	startX := pdf.GetX()

	// 1. Сначала закрашиваем всю полосу от левого края до правого
	pageWidth, _ := pdf.GetPageSize()

	// Сохраняем текущую X позицию
	originalX := startX

	// Закрашиваем всю полосу от левого до правого края
	pdf.SetXY(0, currentY)
	pdf.CellFormat(pageWidth, rowHeight, "", "0", 0, "C", true, 0, "")

	// Возвращаемся к начальной позиции для рисования ячеек
	pdf.SetXY(originalX, currentY)

	// 1. Ячейка №
	pdf.SetTextColor(17, 22, 25)
	pdf.SetFont("Inter", "", 10.5)
	pdf.CellFormat(widths[0], rowHeight, strconv.Itoa(index+1), "1", 0, "C", true, 0, "")

	// 2. Ячейка Наименование (рисуем сначала ячейку, потом текст)
	nameX := startX + widths[0]

	// Рисуем заполненную ячейку
	pdf.SetXY(nameX, currentY)
	pdf.CellFormat(widths[1], rowHeight, "", "1", 0, "C", true, 0, "")

	// Пишем текст в ячейке
	name := picture.GetName()
	desc := picture.GetShortNote()
	if desc == "" {
		desc = "Оборудование системы"
	}

	// Название (верхняя строка) - ОРАНЖЕВЫЙ ЦВЕТ
	pdf.SetXY(nameX+3, currentY+10)
	pdf.SetTextColor(255, 89, 3) // #ff5903 - ОРАНЖЕВЫЙ!
	pdf.SetFont("Inter", "B", 10.5)

	// Используем MultiCell для автоматического переноса названия
	pdf.MultiCell(widths[1]-6, 5, name, "", "L", false)

	// Запоминаем позицию Y после названия
	yAfterName := pdf.GetY()

	// Описание (нижняя строка) - ЧЕРНЫЙ ЦВЕТ
	descStartY := yAfterName + 1 // Ровно 5mm отступа от названия
	pdf.SetXY(nameX+3, descStartY)
	pdf.SetTextColor(17, 22, 25) // #111619 - ЧЕРНЫЙ
	pdf.SetFont("Inter", "", 10.5)

	// Рассчитываем доступную высоту для описания
	availableHeight := rowHeight - (descStartY - currentY) - 3 // Минус отступы

	// Используем MultiCell для описания с ограничением по высоте
	linesNeeded := int(pdf.GetStringWidth(desc) / (widths[1] - 6))
	lineHeight := 4.5
	maxLines := int(availableHeight / lineHeight)

	// Если описание слишком длинное - обрезаем
	if linesNeeded > maxLines {
		// Находим, где обрезать
		words := strings.Fields(desc)
		truncated := ""

		for _, word := range words {
			test := truncated + word + " "
			if pdf.GetStringWidth(test) > (widths[1]-6)*float64(maxLines) {
				truncated = strings.TrimSpace(truncated) + "..."
				break
			}
			truncated = test
		}
		desc = strings.TrimSpace(truncated)
	}

	// Рисуем описание
	pdf.MultiCell(widths[1]-6, lineHeight, desc, "", "L", false)

	// Возвращаемся к правильной позиции для следующих ячеек
	pdf.SetXY(nameX+widths[1], currentY)

	// 3. Кол-во
	count := strconv.Itoa(int(picture.GetCount()))
	pdf.CellFormat(widths[2], rowHeight, count, "1", 0, "C", true, 0, "")

	// 4. Цена
	icon := picture.GetIcon()
	cents := picture.GetMoneyOne()
	pdf.CellFormat(widths[3], rowHeight, fmt.Sprintf("%.2f", float64(cents)/100.0)+icon, "1", 0, "C", true, 0, "")

	// 5. Сумма
	cents = picture.GetMoneyCount()
	pdf.CellFormat(widths[4], rowHeight, fmt.Sprintf("%.2f", float64(cents)/100.0)+icon, "1", 0, "C", true, 0, "")

	// 6. Наличие
	pdf.CellFormat(widths[5], rowHeight, picture.GetPresence(), "1", 0, "C", true, 0, "")

	// 7. Фото
	photoX := pdf.GetX()
	pdf.CellFormat(widths[6], rowHeight, "", "1", 1, "C", true, 0, "")

	// Рисуем фото (или заглушку)
	drawPhotoInCell(pdf, picture.GetImg(), photoX, currentY, widths[6], rowHeight, fillColor)
}

func drawPhotoInCell(pdf *gofpdf.Fpdf, imagePath string, x, y, cellWidth, cellHeight float64, bgColor RGBColor) {
	// Маленькое фото
	photoX := x + (cellWidth-imageSize)/2
	photoY := y + (cellHeight-imageSize)/2

	// Временный отключаем заливку для фото
	pdf.SetFillColor(255, 255, 255)

	//if imagePath != "" {
	//	if _, err := os.Stat(imagePath); err == nil {
	//		pdf.Image(imagePath, photoX, photoY, imageSize, imageSize, false, "", 0, "")
	//		return
	//	}
	//}

	if imagePath != "" {
		if _, err := os.Stat(imagePath); err == nil {
			if name, ok := registerFromFileSmart(pdf, "modelPhoto", imagePath); ok {
				pdf.Image(name, photoX, photoY, imageSize, imageSize, false, "", 0, "")
				return
			}
		}
	}

	// Серый квадрат если нет фото
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.5)
	pdf.Rect(photoX, photoY, imageSize, imageSize, "D")

	// Крестик
	pdf.Line(photoX+2, photoY+2, photoX+imageSize-2, photoY+imageSize-2)
	pdf.Line(photoX+imageSize-2, photoY+2, photoX+2, photoY+imageSize-2)
}

// Функция для добавления подложки (логотипа) на страницы
func addWatermark(pdf *gofpdf.Fpdf) {
	currentPage := pdf.PageNo()

	pageWidth, pageHeight := pdf.GetPageSize()
	forLine := pageHeight - 30

	// Сохраняем всё
	originalTextColorR, originalTextColorG, originalTextColorB := pdf.GetTextColor()
	originalFillColorR, originalFillColorG, originalFillColorB := pdf.GetFillColor()
	originalDrawColorR, originalDrawColorG, originalDrawColorB := pdf.GetDrawColor()
	originalX := pdf.GetX()
	originalY := pdf.GetY()

	// Гарантированно восстанавливаем в конце
	defer func() {
		pdf.SetTextColor(originalTextColorR, originalTextColorG, originalTextColorB)
		pdf.SetFillColor(originalFillColorR, originalFillColorG, originalFillColorB)
		pdf.SetDrawColor(originalDrawColorR, originalDrawColorG, originalDrawColorB)
		pdf.SetXY(originalX, originalY)
	}()

	// Константы для отступов
	const (
		leftMargin  = 32.0
		rightMargin = 32.0
		lineY       = -3.0 // Относительное положение текста от линии
		logoYOffset = 5.0  // Отступ лого от линии
	)

	// Рисуем линию от левого отступа до правого
	pdf.SetDrawColor(255, 89, 3)
	pdf.Line(leftMargin, forLine, pageWidth-rightMargin, forLine)

	// Устанавливаем настройки для вотермарки
	pdf.SetFont("Inter", "", 10.5)

	// Форматируем номер страницы
	currentPageS := fmt.Sprintf("%02d", currentPage)

	// Четная/нечетная страница
	if currentPage%2 == 0 {
		// Четная страница

		// 1. Номер страницы справа (у правого края)
		pdf.SetTextColor(255, 89, 3)
		pageNumX := pageWidth - rightMargin
		pdf.SetXY(pageNumX, forLine+lineY)
		pdf.CellFormat(0, 20, currentPageS, "", 0, "R", false, 0, "")

		// 2. Сайт слева от номера страницы (отступ 50 пунктов)
		pdf.SetTextColor(17, 22, 25)
		siteX := pageNumX - 50
		pdf.SetXY(siteX, forLine+lineY)
		pdf.CellFormat(0, 20, site, "", 0, "L", false, 0, "")

		// 3. Лого слева (у левого края)
		if pdf.GetImageInfo("leftImageIntoWaterMark") != nil {
			pdf.Image("leftImageIntoWaterMark", leftMargin, forLine+logoYOffset, 30, 7, false, "", 0, "")
		}
	} else {
		// Нечетная страница

		// 1. Номер страницы слева (у левого края)
		pdf.SetTextColor(255, 89, 3)
		pdf.SetXY(leftMargin, forLine+lineY)
		pdf.CellFormat(0, 20, currentPageS, "", 0, "L", false, 0, "")

		// 2. Сайт справа от номера страницы
		pdf.SetTextColor(17, 22, 25)
		siteX := leftMargin + 25 // Отступ от номера страницы
		pdf.SetXY(siteX, forLine+lineY)
		pdf.CellFormat(0, 20, site, "", 0, "L", false, 0, "")

		// 3. Лого справа (у правого края)
		if pdf.GetImageInfo("rightImageIntoWaterMark") != nil {
			pdf.Image("rightImageIntoWaterMark", pageWidth-rightMargin-30, forLine+logoYOffset, 30, 5, false, "", 0, "")
		}
	}
}

// Страница заключения
func createConclusionPage(pdf *gofpdf.Fpdf) {
	// Очищаем текущую страницу
	pdf.SetY(50)

	// Заголовок
	pdf.SetFont("montserrat", "B", 24)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 20, "Заключение", "", 1, "C", false, 0, "")

	pdf.Ln(20)

	// Текст заключения
	pdf.SetFont("montserrat", "", 14)
	pdf.SetTextColor(0, 0, 0)

	text := "Настоящий отчет содержит информацию о всех моделях, находящихся в базе данных. " +
		"Каждая модель представлена с фотографией, техническими характеристиками и статусом наличия. " +
		"Отчет был сгенерирован автоматически и содержит актуальную информацию на момент создания."

	// Используем MultiCell для правильного переноса текста
	pdf.SetX(20)
	pdf.MultiCell(257, 10, text, "", "L", false)

	pdf.Ln(30)

	// Благодарность
	pdf.SetFont("montserrat", "I", 12)
	pdf.SetX(0)
	pdf.CellFormat(0, 10, "Благодарим за использование нашего сервиса!", "", 1, "C", false, 0, "")
}

func getImageType(imageData []byte) string {
	if len(imageData) < 12 {
		return ""
	}

	// Проверяем сигнатуры разных форматов
	switch {
	// JPEG: начинается с FF D8 FF
	case bytes.HasPrefix(imageData, []byte{0xFF, 0xD8, 0xFF}):
		return "jpg"
	// PNG: начинается с 89 50 4E 47 0D 0A 1A 0A
	case bytes.HasPrefix(imageData, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}):
		return "png"
	// GIF: начинается с "GIF87a" или "GIF89a"
	case bytes.HasPrefix(imageData, []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}) ||
		bytes.HasPrefix(imageData, []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}):
		return "gif"
	default:
		return ""
	}
}

func setDefaultBackground(pdf *gofpdf.Fpdf) {
	pdf.SetFillColor(41, 128, 185)
	pdf.Rect(0, 0, 297, 210, "F")
}

func setSpecificationEquipment(pdf *gofpdf.Fpdf, idCompany, number int) {
	switch idCompany {
	case 2:
		pdf.SetXY(25, 0)
		pdf.SetFont("montserrat", "", 74)
		pdf.SetTextColor(255, 89, 3)
		pdf.CellFormat(50, 50, strconv.Itoa(number), "", 0, "L", false, 0, "")

		// Заголовок
		pdf.SetFont("montserrat", "", 28)
		pdf.SetTextColor(17, 22, 25)
		pdf.SetXY(50, 13.5)
		pdf.MultiCell(200, 10, "СПЕЦИФИКАЦИЯ\nОБОРУДОВАНИЯ", "", "L", false)
		break
	case 3:
		pdf.SetXY(25, 0)
		pdf.SetFont("montserrat", "", 74)
		pdf.SetTextColor(255, 89, 3)
		pdf.CellFormat(50, 50, strconv.Itoa(number), "", 0, "L", false, 0, "")

		pdf.SetFont("montserrat", "", 28)
		pdf.SetTextColor(17, 22, 25)
		pdf.SetXY(50, 13.5)
		pdf.MultiCell(200, 10, "СПЕЦИФИКАЦИЯ\nОБОРУДОВАНИЯ", "", "L", false)
		break
	}
}

func setImageIntoPDF(pdf *gofpdf.Fpdf, imageData []byte, positionX, positionY, width, height float64, nameImage string, withDefault bool) {
	if len(imageData) == 0 {
		if withDefault {
			setDefaultBackground(pdf)
		}
		return
	}

	// Пытаемся определить тип изображения
	imageType := getImageType(imageData)

	// Если тип PNG или не определен (возможно PNG 16-bit), пробуем конвертировать
	if imageType == "" || imageType == "png" || imageType == "PNG" {
		if convertedData, err := convertPNG16to8(imageData); err == nil && len(convertedData) > 0 {
			// Проверяем, что конвертация дала валидный PNG
			if newType := getImageType(convertedData); newType == "png" {
				imageData = convertedData
				imageType = "png"
			}
		}
	}

	// Если тип все еще не определен
	if imageType == "" {
		if withDefault {
			setDefaultBackground(pdf)
		}
		return
	}

	reader := bytes.NewReader(imageData)
	imgInfo := pdf.RegisterImageReader(nameImage, imageType, reader)
	if imgInfo == nil {
		if withDefault {
			setDefaultBackground(pdf)
		}
		return
	}

	pdf.Image(nameImage, positionX, positionY, width, height, false, "", 0, "")
}

func createCharacteristic(pdf *gofpdf.Fpdf, additionallyEquipment []*createpdffile.Models, characteristicData *createpdffile.ModelsData, number int) int {
	startPage := pdf.PageNo()
	var endPage int
	pdf.SetLeftMargin(32)

	// Заголовок раздела
	pdf.SetY(17)
	pdf.SetFont("montserrat", "", 24)
	pdf.SetTextColor(237, 114, 3)
	pdf.CellFormat(0, 24*math, strconv.Itoa(number)+".1", "", 0, "L", false, 0, "")
	pdf.SetTextColor(37, 36, 36)
	pdf.SetXY(34, 17)
	pdf.CellFormat(0, 24*math, strings.Repeat(" ", 5)+"Характеристики системы", "", 0, "L", false, 0, "")

	// Первая группа: Мощностные характеристики
	yPos := 17 + 24*math + 22.576
	pdf.SetY(yPos)
	pdf.SetFont("inter", "", 12)
	pdf.SetTextColor(237, 114, 3)
	pdf.CellFormat(0, 16*math, "Мощностные характеристики системы", "", 0, "L", false, 0, "")

	// Данные первой группы
	yPos += 16 * math
	pdf.SetY(yPos)
	pdf.SetFont("inter", "", 12)
	pdf.SetTextColor(37, 36, 36)

	labelWidth := (32+99.3868)*math + 100
	valueWidth := 60.0 * math

	powerData1 := []PowerData{
		{"Общая потребляемая мощность, Вт", characteristicData.GetPowerConsumption()},
		{"Общая выходная мощность, Вт", characteristicData.GetMaxPower()},
		{"Общая мощность громко говорителей, Вт", characteristicData.GetRatedPowerSpeaker()},
	}

	powerDataPDF(pdf, powerData1, labelWidth, valueWidth)

	// Отступ между группами - 19.754 мм
	yPos = pdf.GetY() + 19.754
	pdf.SetY(yPos)

	// Вторая группа: Массогабаритные характеристики
	pdf.SetFont("inter", "", 12)
	pdf.SetTextColor(237, 114, 3)
	pdf.MultiCell(0, 16*math, "Массогабаритные характеристики\nоборудования по спецификации", "", "L", false)

	// Данные второй группы
	yPos = pdf.GetY() + 5*math // Небольшой отступ после заголовка
	pdf.SetY(yPos)
	pdf.SetFont("inter", "", 12)
	pdf.SetTextColor(37, 36, 36)

	powerData2 := []PowerData{
		{"Общая высота, U", characteristicData.GetUnit()},
		{"Масса брутто, кг", characteristicData.GetMass()},
		{"Объем с учетом упаковки, м3", characteristicData.GetSize()},
	}

	powerDataPDF(pdf, powerData2, labelWidth, valueWidth)

	endPage = pdf.PageNo()

	addWatermark(pdf)
	return endPage - startPage
}

func powerDataPDF(pdf *gofpdf.Fpdf, powerData []PowerData, labelWidth, valueWidth float64) {
	for _, data := range powerData {
		x := pdf.GetX()
		pdf.CellFormat(labelWidth, 16*math, data.Label, "", 0, "L", false, 0, "")
		pdf.SetX(x + labelWidth)

		var valueText string
		switch v := data.Value.(type) {
		case int:
			valueText = strconv.Itoa(v)
		case float64:
			valueText = strconv.FormatFloat(v, 'f', -1, 64)
		case float32:
			valueText = strconv.FormatFloat(float64(v), 'f', -1, 32)
		default:
			valueText = fmt.Sprintf("%v", v)
		}

		pdf.CellFormat(valueWidth, 16*math, valueText, "", 0, "L", false, 0, "")

		if data != powerData[len(powerData)-1] {
			pdf.Ln(16 * math)
		}
	}
}

func hashName(prefix string, b []byte) string {
	h := sha1.Sum(b)
	return prefix + "_" + hex.EncodeToString(h[:8])
}

func normalizeToPNG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	img = imaging.Clone(img)

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func registerSmart(pdf *gofpdf.Fpdf, prefix string, raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}

	t := getImageType(raw)

	// Если это настоящий JPEG — оставляем JPEG
	if t == "jpg" {
		name := hashName(prefix, raw)
		if pdf.RegisterImageReader(name, "jpg", bytes.NewReader(raw)) != nil {
			return name, true
		}
		return "", false
	}

	// Если PNG — оставляем PNG
	if t == "png" {
		name := hashName(prefix, raw)
		if pdf.RegisterImageReader(name, "png", bytes.NewReader(raw)) != nil {
			return name, true
		}
		return "", false
	}

	// Иначе — декодируем и перекодируем в PNG
	pngData, err := normalizeToPNG(raw)
	if err != nil {
		return "", false
	}
	name := hashName(prefix, pngData)
	if pdf.RegisterImageReader(name, "png", bytes.NewReader(pngData)) != nil {
		return name, true
	}
	return "", false
}

func registerFromFileSmart(pdf *gofpdf.Fpdf, prefix, path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil || len(b) == 0 {
		return "", false
	}

	return registerSmart(pdf, prefix, b)
}

func createSystemFeatures(pdf *gofpdf.Fpdf, features []*createpdffile.SystemFeatures, number int) {
	pdf.SetXY(25, 0)
	pdf.SetFont("montserrat", "", 74)
	pdf.SetTextColor(255, 89, 3)
	pdf.CellFormat(50, 50, strconv.Itoa(number), "", 0, "L", false, 0, "")

	// Заголовок
	pdf.SetFont("montserrat", "", 28)
	pdf.SetTextColor(17, 22, 25)
	pdf.SetXY(50, 13.5)
	pdf.MultiCell(200, 10, "ОСОБЕННОСТИ СИСТЕМЫ И\nТРЕБОВАНИЯ ЗАКАЗЧИКА", "", "L", false)

	leftPad := 32.0
	rightPad := 32.0
	startY := 60.0

	items := make([]string, 0, len(features))
	for _, f := range features {
		items = append(items, f.GetFeature())
	}

	drawNumberedListWordLike(pdf, items, leftPad, rightPad, startY)

	addWatermark(pdf)
}

// drawNumberedListWordLike печатает нумерованный список в стиле Word:
// - общий блок: leftPad..(pageW-rightPad)
// - номер с небольшим доп. отступом (numIndent)
// - переносы начинаются от левого края блока (под номером тоже)
func drawNumberedListWordLike(pdf *gofpdf.Fpdf, items []string, leftPad, rightPad, startY float64) {
	pageW, _ := pdf.GetPageSize()
	blockW := pageW - leftPad - rightPad

	// Шрифт/цвет для списка
	pdf.SetFont("inter", "", 10.5)
	pdf.SetTextColor(17, 22, 25)

	lineH := 6.0 // подберите под ваш шрифт

	// "Отступ номера" внутри блока (1.25 мм ≈ 1.25 * 72 / 25.4 = 3.54 pt)
	// В gofpdf единицы по умолчанию мм, так что 1.25 — это 1.25 мм.
	numIndent := 1.25

	pdf.SetXY(leftPad, startY)

	for i, txt := range items {
		n := i + 1

		// Формируем строку: [внутренний отступ] + "1." + пробел + текст
		// Переносы MultiCell будут начинаться от leftPad (то есть могут быть под номером).
		prefix := strconv.Itoa(n) + "." + " "
		line := strings.Repeat(" ", 10) + prefix + txt

		// Внутренний отступ реализуем сдвигом X перед печатью, но ширину блока уменьшаем,
		// чтобы правое поле оставалось ровно 32.
		x := leftPad + numIndent
		y := pdf.GetY()
		pdf.SetXY(x, y)
		pdf.MultiCell(blockW-numIndent, lineH, line, "", "L", false)

		pdf.SetX(leftPad)
		pdf.Ln(1.5)
	}
}

func splitByRuneLimitWordBoundary(s string, limit int) (left, right string) {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= limit || limit <= 0 {
		return string(rs), ""
	}

	// 1) Базовый cut: последнее разделяющее место до лимита (пробел/дефисы)
	cut := lastSeparatorCut(rs, limit)

	// Если всё равно что-то пошло не так — запасной вариант
	if cut <= 0 || cut >= len(rs) {
		left = strings.TrimSpace(string(rs[:min(limit, len(rs))]))
		right = strings.TrimSpace(string(rs[min(limit, len(rs)):]))
		return
	}

	// 2) Проверяем: попали ли мы "внутрь предложения"
	// Предложение считаем по . ! ? … (можно расширить при необходимости)
	sStart := sentenceStart(rs, cut)
	sEnd := sentenceEnd(rs, cut)

	inSentence := (sStart < cut && cut < sEnd)

	if inSentence {
		// Сколько слов осталось до конца предложения справа
		remainWords := countWords(rs[cut:sEnd])

		// 2a) Если до конца предложения осталось 1–2 слова — дописываем их в левую
		if remainWords <= 2 {
			// Разрешаем небольшое превышение лимита ради 1–2 слов
			cut = sEnd
		} else {
			// 2b) Иначе стараемся перенести предложение целиком в правую:
			// cut -> начало предложения
			// Но если начало предложения слишком рано (левая станет пустой/почти пустой),
			// то оставляем старый cut.
			newCut := sStart
			if newCut > 0 {
				leftCandidate := strings.TrimSpace(string(rs[:newCut]))
				// "почти пустая" — на практике удобно проверять по числу слов
				// (порог можно поменять: 0/1/2 слова).
				if countWords([]rune(leftCandidate)) >= 3 {
					cut = newCut
				}
			}

			// 2c) Если предложение само по себе длиннее лимита (не уместить целиком),
			// то переносить "целиком" бессмысленно — режем по словам как раньше.
			if (sEnd - sStart) > limit {
				cut = lastSeparatorCut(rs, limit)
			}
		}
	}

	left = strings.TrimSpace(string(rs[:cut]))
	right = strings.TrimSpace(string(rs[cut:]))
	return
}

func lastSeparatorCut(rs []rune, limit int) int {
	if limit >= len(rs) {
		return len(rs)
	}
	cut := limit
	for i := limit; i > 0; i-- {
		r := rs[i-1]
		if unicode.IsSpace(r) || r == '-' || r == '—' {
			cut = i
			break
		}
	}
	// Если до лимита вообще нет разделителей — режем строго по лимиту
	if cut == 0 {
		cut = limit
	}
	return cut
}

func sentenceStart(rs []rune, pos int) int {
	// Ищем последнюю конечную пунктуацию ДО pos, старт предложения = следующий символ
	for i := pos - 1; i >= 0; i-- {
		if isSentenceEnd(rs[i]) {
			return i + 1
		}
	}
	return 0
}

func sentenceEnd(rs []rune, pos int) int {
	// Ищем ближайшую конечную пунктуацию ПОСЛЕ pos
	for i := pos; i < len(rs); i++ {
		if isSentenceEnd(rs[i]) {
			return i + 1
		}
	}
	return len(rs)
}

func isSentenceEnd(r rune) bool {
	switch r {
	case '.', '!', '?', '…':
		return true
	default:
		return false
	}
}

func countWords(rs []rune) int {
	inWord := false
	cnt := 0
	for _, r := range rs {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				inWord = true
				cnt++
			}
		} else if unicode.IsSpace(r) {
			inWord = false
		} else {
			// пунктуация/прочее — считаем как разделитель слова
			inWord = false
		}
	}
	return cnt
}

func selectsEquipments(pdf *gofpdf.Fpdf, text string, number int) {
	// Большая цифра
	pdf.SetXY(25, 0)
	pdf.SetFont("montserrat", "", 74)
	pdf.SetTextColor(255, 89, 3)
	pdf.CellFormat(50, 50, strconv.Itoa(number), "", 0, "L", false, 0, "")

	// Заголовок
	pdf.SetFont("montserrat", "", 28)
	pdf.SetTextColor(17, 22, 25)
	pdf.SetXY(50, 13.5)
	pdf.MultiCell(200, 10, "ВЫБОР\nОБОРУДОВАНИЯ", "", "L", false)

	// --- ТЕКСТ В 2 КОЛОНКИ ---

	// Параметры колонок
	leftPad := 32.0
	rightPad := 32.0
	gutter := 12.0 // расстояние между колонками (подберите под макет)
	startY := 60.0 // начало текста под заголовком (подберите под ваш макет)
	lineH := 5.2   // высота строки для Inter 10.5 (подберите при необходимости)

	pageW, _ := pdf.GetPageSize()
	usableW := pageW - leftPad - rightPad
	colW := (usableW - gutter) / 2.0

	leftX := leftPad
	rightX := leftPad + colW + gutter

	// Режем текст: 950 символов в левую колонку, остаток — в правую (до 950)
	leftText, rest := splitByRuneLimitWordBoundary(text, 950)
	rightText, _ := splitByRuneLimitWordBoundary(rest, 950)

	// Шрифт текста
	pdf.SetFont("inter", "", 10.5)
	pdf.SetTextColor(17, 22, 25)

	// Левая колонка
	pdf.SetXY(leftX, startY)
	pdf.MultiCell(colW, lineH, leftText, "", "J", false) // "J" если нужно как в Word (по ширине)

	// Правая колонка (с той же высоты startY)
	pdf.SetXY(rightX, startY)
	pdf.MultiCell(colW, lineH, rightText, "", "J", false)
}

func imageEquipments(pdf *gofpdf.Fpdf, picture []byte, name, nameImage string, number, subNumber int) {
	positionX := 32
	pdf.SetLeftMargin(float64(positionX))

	// Заголовок раздела
	pdf.SetY(17)
	pdf.SetFont("montserrat", "", 24)
	pdf.SetTextColor(237, 114, 3)
	pdf.CellFormat(0, 24*math, strconv.Itoa(number)+"."+strconv.Itoa(subNumber), "", 0, "L", false, 0, "")
	pdf.SetTextColor(37, 36, 36)
	pdf.SetXY(float64(positionX+2), 17)
	pdf.CellFormat(0, 24*math, strings.Repeat(" ", 5)+name, "", 0, "L", false, 0, "")

	_, pageHeight := pdf.GetPageSize()
	positionY := 25

	// 30 -- Высота для waterMark
	setImageIntoPDF(pdf, picture, float64(positionX), float64(positionY), 233, pageHeight-30-float64(positionY)-10, nameImage, true)

	addWatermark(pdf)
}
