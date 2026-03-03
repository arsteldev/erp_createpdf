package createPDF

import (
	"bytes"
	"fmt"
	createpdffile "github.com/arsteldev/createPDF/proto"
	"github.com/disintegration/imaging"
	"github.com/jung-kurt/gofpdf"
	"image/png"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	FullInformationImageWidthArstel  = 135
	FullInformationImageHeightArstel = 47
)

func Text(pdf *gofpdf.Fpdf, rgb RGBColor, font Font, position Position, cellString MultiCellString) {
	pdf.SetFont(font.font, font.style, font.size)
	pdf.SetTextColor(rgb.R, rgb.G, rgb.B)

	switch {
	case position.X == -1 && position.Y == -1:
	case position.X != -1 && position.Y == -1:
		pdf.SetX(position.X)
	case position.X == -1 && position.Y != -1:
		pdf.SetY(position.Y)
	default:
		pdf.SetXY(position.X, position.Y)
	}

	x := pdf.GetX()
	pdf.MultiCell(cellString.w, cellString.h, cellString.txtStr, cellString.borderStr, cellString.alignStr, cellString.fill)
	pdf.SetX(x)
}

func TextCellFormat(pdf *gofpdf.Fpdf, rgb RGBColor, font Font, position Position, cellString CellString) {
	if position.X == -1 && position.Y == -1 {
	} else if position.X == -1 && position.Y != -1 {
		pdf.SetY(position.Y)
	} else if position.X != -1 && position.Y != -1 {
		pdf.SetXY(position.X, position.Y)
	}

	pdf.SetFont(font.font, font.style, font.size)
	pdf.SetTextColor(rgb.R, rgb.G, rgb.B)
	pdf.CellFormat(cellString.w, cellString.h, cellString.txtStr, cellString.borderStr, cellString.ln, cellString.alignStr, cellString.fill, cellString.link, cellString.linkStr)
}

func SetImageIntoPDF(pdf *gofpdf.Fpdf, imageData []byte, positionX, positionY, width, height float64, nameImage string, withDefault bool) {
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

func setDefaultBackground(pdf *gofpdf.Fpdf) {
	pdf.SetFillColor(41, 128, 185)
	pdf.Rect(0, 0, 297, 210, "F")
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

func DrawTableHeader(pdf *gofpdf.Fpdf, widths []float64, headers []string, headerHeights float64, full bool, font Font, color RGBColor) {
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont(font.font, font.style, font.size)
	// Для шапки делаем границы того же цвета что и фон
	pdf.SetDrawColor(color.R, color.G, color.B)

	pageWidth, _ := pdf.GetPageSize()
	currentY := pdf.GetY()
	startX := pdf.GetX()

	// 1. Сначала закрашиваем всю полосу от левого края до правого
	if full {
		drawBackgroud(pdf, Position{X: 0, Y: currentY}, Parametrs{Width: pageWidth, Height: headerHeights}, color)
	} else {
		leftMargin := 20.25
		pageWidth = pageWidth - (2 * leftMargin)
		drawBackgroud(pdf, Position{X: leftMargin, Y: 39.533}, Parametrs{Width: pageWidth, Height: headerHeights}, color)
	}

	currentY = pdf.GetY()

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
		log.Printf("headers: %s, X: %f", header, x)
		drawHeaderCell(x, currentY, widths[i], headerHeights, header, "L")
		x += widths[i]
	}

	// Устанавливаем позицию для следующей строки
	pdf.SetXY(startX, currentY+headerHeights)
}

//func DrawTableRow(pdf *gofpdf.Fpdf, index int, columns []Column, widths []float64, isEvenRow bool) {
//	// Определяем цвет фона строки
//	var fillColor RGBColor
//	if isEvenRow {
//		fillColor = RGBColor{232, 237, 237} // #e8eded - серая строка
//	} else {
//		fillColor = RGBColor{255, 255, 255} // #ffffff - белая строка
//	}
//
//	// Устанавливаем цвет заливки и границы
//	pdf.SetFillColor(fillColor.R, fillColor.G, fillColor.B)
//	pdf.SetDrawColor(fillColor.R, fillColor.G, fillColor.B) // Границы того же цвета!
//	pdf.SetLineWidth(0.1)
//
//	currentY := pdf.GetY()
//	startX := pdf.GetX()
//
//	// 1. Сначала закрашиваем всю полосу от левого края до правого
//	pageWidth, _ := pdf.GetPageSize()
//
//	// Сохраняем текущую X позицию
//	originalX := startX
//
//	// Закрашиваем всю полосу от левого до правого края
//	pdf.SetXY(0, currentY)
//	pdf.CellFormat(pageWidth, rowHeight, "", "0", 0, "C", true, 0, "")
//
//	// Возвращаемся к начальной позиции для рисования ячеек
//	pdf.SetXY(originalX, currentY)
//
//	// 1. Ячейка №
//	pdf.SetTextColor(17, 22, 25)
//	pdf.SetFont("Inter", "", 10.5)
//	pdf.CellFormat(widths[0], rowHeight, strconv.Itoa(index+1), "1", 0, "C", true, 0, "")
//
//	// 2. Ячейка Наименование (рисуем сначала ячейку, потом текст)
//	nameX := startX + widths[0]
//
//	// Рисуем заполненную ячейку
//	pdf.SetXY(nameX, currentY)
//	pdf.CellFormat(widths[1], rowHeight, "", "1", 0, "C", true, 0, "")
//
//	// Пишем текст в ячейке
//	name := picture.GetName()
//	desc := picture.GetShortNote()
//	if desc == "" {
//		desc = "Оборудование системы"
//	}
//
//	// Название (верхняя строка) - ОРАНЖЕВЫЙ ЦВЕТ
//	pdf.SetXY(nameX+3, currentY+10)
//	pdf.SetTextColor(255, 89, 3) // #ff5903 - ОРАНЖЕВЫЙ!
//	pdf.SetFont("Inter", "B", 10.5)
//
//	// Используем MultiCell для автоматического переноса названия
//	pdf.MultiCell(widths[1]-6, 5, name, "", "L", false)
//
//	// Запоминаем позицию Y после названия
//	yAfterName := pdf.GetY()
//
//	// Описание (нижняя строка) - ЧЕРНЫЙ ЦВЕТ
//	descStartY := yAfterName + 1 // Ровно 5mm отступа от названия
//	pdf.SetXY(nameX+3, descStartY)
//	pdf.SetTextColor(17, 22, 25) // #111619 - ЧЕРНЫЙ
//	pdf.SetFont("Inter", "", 10.5)
//
//	// Рассчитываем доступную высоту для описания
//	availableHeight := rowHeight - (descStartY - currentY) - 3 // Минус отступы
//
//	// Используем MultiCell для описания с ограничением по высоте
//	linesNeeded := int(pdf.GetStringWidth(desc) / (widths[1] - 6))
//	lineHeight := 4.5
//	maxLines := int(availableHeight / lineHeight)
//
//	// Если описание слишком длинное - обрезаем
//	if linesNeeded > maxLines {
//		// Находим, где обрезать
//		words := strings.Fields(desc)
//		truncated := ""
//
//		for _, word := range words {
//			test := truncated + word + " "
//			if pdf.GetStringWidth(test) > (widths[1]-6)*float64(maxLines) {
//				truncated = strings.TrimSpace(truncated) + "..."
//				break
//			}
//			truncated = test
//		}
//		desc = strings.TrimSpace(truncated)
//	}
//
//	// Рисуем описание
//	pdf.MultiCell(widths[1]-6, lineHeight, desc, "", "L", false)
//
//	// Возвращаемся к правильной позиции для следующих ячеек
//	pdf.SetXY(nameX+widths[1], currentY)
//
//	// 3. Кол-во
//	count := strconv.Itoa(int(picture.GetCount()))
//	pdf.CellFormat(widths[2], rowHeight, count, "1", 0, "C", true, 0, "")
//
//	// 4. Цена
//	icon := picture.GetIcon()
//	cents := picture.GetMoneyOne()
//	pdf.CellFormat(widths[3], rowHeight, fmt.Sprintf("%.2f", float64(cents)/100.0)+icon, "1", 0, "C", true, 0, "")
//
//	// 5. Сумма
//	cents = picture.GetMoneyCount()
//	pdf.CellFormat(widths[4], rowHeight, fmt.Sprintf("%.2f", float64(cents)/100.0)+icon, "1", 0, "C", true, 0, "")
//
//	// 6. Наличие
//	pdf.CellFormat(widths[5], rowHeight, picture.GetPresence(), "1", 0, "C", true, 0, "")
//
//	// 7. Фото
//	photoX := pdf.GetX()
//	pdf.CellFormat(widths[6], rowHeight, "", "1", 1, "C", true, 0, "")
//
//	// Рисуем фото (или заглушку)
//	drawPhotoInCell(pdf, picture.GetImg(), photoX, currentY, widths[6], rowHeight, fillColor)
//}

func DrawTableRows(pdf *gofpdf.Fpdf, rows [][]Row, headerHeight, rowHeight, leftMargin float64, firstPageRows, nextPageRows int) {
	if pdf.GetY() < headerHeight+0.1 {
		pdf.SetY(headerHeight + 0.1)
	}

	log.Printf("SetY %f", pdf.GetY())

	firstPage := true
	rowsOnPage := 0

	for i, row := range rows {
		limit := nextPageRows
		if firstPage {
			limit = firstPageRows
		}

		// Переход на новую страницу
		if rowsOnPage >= limit {
			AddWatermark(pdf)
			pdf.AddPage()

			firstPage = false
			rowsOnPage = 0

			// ВАЖНО: на новой странице снова ставим Y под шапку + отступ
			pdf.SetY(((headerHeight * 2) - 1) + 0.1)
		}

		// Чередование фона
		bg := RGBColor{255, 255, 255}
		//bg := RGBColor{232, 237, 237}
		if i%2 == 1 {
			//bg = RGBColor{255, 255, 255}
			bg = RGBColor{232, 237, 237}
		}

		// Рисуем строку строго по текущему курсору PDF
		pos := Position{X: pdf.GetX(), Y: pdf.GetY()}
		drawRow(pdf, row, pos, bg, rowHeight, leftMargin)

		rowsOnPage++
	}
}

// drawRow — рисует одну строку таблицы (фон + ячейки)
func drawRow(pdf *gofpdf.Fpdf, columns []Row, position Position, color RGBColor, rowHeight1, leftMargin float64) {
	// Ширина страницы
	pageWidth, _ := pdf.GetPageSize()

	// Фон строки
	drawBackgroud(pdf, Position{X: leftMargin, Y: position.Y}, Parametrs{Width: pageWidth, Height: rowHeight1}, color)

	// Текст по ячейкам
	x := position.X
	y := position.Y
	pdf.SetXY(x, y)

	for _, c := range columns {
		pdf.SetXY(x, y)
		pdf.SetTextColor(0, 0, 0)
		// ВАЖНО: используйте MultiCell, если возможны переносы по строкам
		pdf.CellFormat(c.Width, rowHeight1, c.Text, "", 0, "L", false, 0, "")
		x += c.Width
	}

	// Переход на следующую строку
	pdf.SetXY(position.X, position.Y+rowHeight1)
}

func drawBackgroud(pdf *gofpdf.Fpdf, position Position, parametrs Parametrs, color RGBColor) {
	// Выбираем цвет для заливки
	pdf.SetFillColor(color.R, color.G, color.B)
	// Закрашиваем всю полосу
	pdf.SetXY(position.X, position.Y)
	pdf.CellFormat(parametrs.Width, parametrs.Height, "", "0", 0, "C", true, 0, "")
}

func AddWatermark(pdf *gofpdf.Fpdf) {
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
	if currentPage%2 == 1 {
		// Нечетная страница

		// 1. Номер страницы справа (у правого края)
		pdf.SetTextColor(255, 89, 3)
		pageNumX := pageWidth - rightMargin
		pdf.SetXY(pageNumX, forLine+lineY)
		pdf.CellFormat(0, 20, currentPageS, "", 0, "R", false, 0, "")

		// 2. Сайт слева от номера страницы (отступ 50 пунктов)
		pdf.SetTextColor(17, 22, 25)
		//siteX := pageNumX - 50
		siteX := pageNumX - 40
		pdf.SetXY(siteX, forLine+lineY)
		pdf.CellFormat(0, 20, site, "", 0, "L", false, 0, "")

		// 3. Лого слева (у левого края)
		if pdf.GetImageInfo("leftImageIntoWaterMark") != nil {
			pdf.Image("leftImageIntoWaterMark", leftMargin, forLine+logoYOffset, 30, 7, false, "", 0, "")
		}
	} else {
		// Четная страница

		// 1. Номер страницы слева (у левого края)
		pdf.SetTextColor(255, 89, 3)
		pdf.SetXY(leftMargin, forLine+lineY)
		pdf.CellFormat(0, 20, currentPageS, "", 0, "L", false, 0, "")

		// 2. Сайт справа от номера страницы
		pdf.SetTextColor(17, 22, 25)
		//siteX := leftMargin + 25 // Отступ от номера страницы
		siteX := leftMargin + 25 // Отступ от номера страницы
		pdf.SetXY(siteX, forLine+lineY)
		pdf.CellFormat(0, 20, site, "", 0, "L", false, 0, "")

		// 3. Лого справа (у правого края)
		if pdf.GetImageInfo("rightImageIntoWaterMark") != nil {
			pdf.Image("rightImageIntoWaterMark", pageWidth-rightMargin-30, forLine+logoYOffset, 30, 5, false, "", 0, "")
		}
	}
}

func AddWatermarkRondo(pdf *gofpdf.Fpdf) {
	currentPage := pdf.PageNo()

	pageWidth, pageHeight := pdf.GetPageSize()
	forLine := pageHeight - 24.659

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
		leftMargin  = 20.4
		rightMargin = 20.4
		//lineY       = -3.0 // Относительное положение текста от линии
		//logoYOffset = 5.0  // Отступ лого от линии
	)

	// Рисуем линию от левого отступа до правого
	pdf.SetDrawColor(greenColor.R, greenColor.G, greenColor.B)
	pdf.Line(leftMargin, forLine, pageWidth-rightMargin, forLine)

	// Устанавливаем настройки для вотермарки
	pdf.SetFont("Inter", "", 10.5)

	// Форматируем номер страницы
	var (
		currentPageS string
		posX         float64
	)
	if currentPage > 9 {
		currentPageS = fmt.Sprintf("%02d", currentPage)
		posX = 271.5
	} else {
		currentPageS = fmt.Sprintf("%2d", currentPage)
		posX = 272.5
	}

	//Четная/нечетная страница, меняется только картинка. Остальное остается как было
	if currentPage%2 == 0 {
		// Rondo
		if pdf.GetImageInfo("logoRondo") != nil {
			pdf.Image("logoRondo", leftMargin, 188.395, 10.978, 11.8, false, "", 0, "")
		}
	} else {
		// LDA
		if pdf.GetImageInfo("logoLDA") != nil {
			pdf.Image("logoLDA", leftMargin, 190.045, 26.67, 9.73, false, "", 0, "")
		}
	}

	// Сайт
	TextCellFormat(
		pdf,
		greenColor,
		Font{"Inter", "", 10.5},
		Position{63.443, 185.397},
		CellString{0, 20, site, "", 0, "L", false, 0, ""},
	)

	// № Страницы
	TextCellFormat(
		pdf,
		greenColor,
		Font{"Inter", "", 10.5},
		Position{posX, 185.397},
		CellString{0, 20, currentPageS, "", 0, "L", false, 0, ""},
	)
}

func CreatePageSpecials(pdf *gofpdf.Fpdf) {
	pageWidth, pageHeight := pdf.GetPageSize()
	pdf.Image("coverBackground", 0, 0, pageWidth, pageHeight, false, "", 0, "")
	pdf.Image("companyLogo", 34, 17, 60, 15, false, "", 0, "")
	pdf.Image("smallLogo", pageWidth-64, 19, 32, 9, false, "", 0, "")
	pdf.Image("mainPageLogo", pageWidth-120, pageHeight-130, 110, 50, false, "", 0, "")

	Text(
		pdf,
		RGBColor{R: 37, G: 36, B: 36},
		Font{font: "montserrat", style: "M", size: 34},
		Position{X: -1, Y: 47},
		MultiCellString{w: 0, h: 38 * math, txtStr: "ОПИСАНИЕ\nОБОРУДОВАНИЯ", borderStr: "", alignStr: "L", fill: false},
	)

	Text(
		pdf,
		RGBColor{R: 37, G: 36, B: 36},
		Font{font: "montserrat", style: "M", size: 14},
		Position{X: -1, Y: 87},
		MultiCellString{w: 0, h: 10.8 * math, txtStr: "Приложение к коммерческому\nпредложению", borderStr: "", alignStr: "L", fill: false},
	)
}

func CreatePageFullInformation(pdf *gofpdf.Fpdf, model *createpdffile.Models) {
	pageWidth, _ := pdf.GetPageSize()
	mainColor := RGBColor{R: 17, G: 22, B: 25}
	headerColor := RGBColor{R: 255, G: 89, B: 3}
	interRegular18 := Font{font: "inter", style: "", size: 18}
	interRegular10 := Font{font: "inter", style: "", size: 10}
	interRegular8 := Font{font: "inter", style: "", size: 8}
	monseratMedium16 := Font{font: "montserrat", style: "M", size: 16}

	pdf.Image("appendix", pageWidth-20, 0, 20, 210, false, "", 0, "")
	pdf.Image(model.Name, 140, 30, FullInformationImageWidthArstel, FullInformationImageHeightArstel, false, "", 0, "")

	TextCellFormat(
		pdf,
		headerColor,
		Font{
			font:  "montserrat",
			style: "M",
			size:  36,
		},
		Position{
			X: 30,
			Y: 30,
		},
		CellString{0, 14.4 * math, model.Name, "", 1, "L", false, 0, ""},
	)
	lineX := float64(30)
	lineY := float64(80)

	Text(pdf, mainColor, interRegular18, Position{X: lineX, Y: 40}, MultiCellString{w: 100, h: 14.4 * math, txtStr: model.ShortNote, borderStr: "", alignStr: "L", fill: false})
	Text(pdf, headerColor, monseratMedium16, Position{X: lineX, Y: lineY}, MultiCellString{w: 40, h: 14.4 * math, txtStr: "ОПИСАНИЕ", borderStr: "", alignStr: "J", fill: false})
	Text(pdf, mainColor, interRegular8, Position{X: lineX, Y: lineY + 10}, MultiCellString{w: 100, h: 14.4 * math, txtStr: model.Description, borderStr: "", alignStr: "J", fill: false})

	Text(pdf, headerColor, monseratMedium16, Position{X: 150, Y: lineY}, MultiCellString{w: 120, h: 14.4 * math, txtStr: "ОСОБЕННОСТИ", borderStr: "", alignStr: "J", fill: false})
	startX := 150
	startY := 90
	for _, special := range model.Specials {
		Text(pdf, mainColor, interRegular10, Position{X: float64(startX), Y: float64(startY)}, MultiCellString{w: 100, h: 8 * math, txtStr: "* " + special.Special, borderStr: "", alignStr: "J", fill: false})
		pdf.Ln(2)
		startY = -1
	}

	Text(pdf, headerColor, monseratMedium16, Position{X: lineX, Y: 140}, MultiCellString{w: 140, h: 14.4 * math, txtStr: "ТЕХНИЧЕСКИЕ ХАРАКТЕРИСТИКИ", borderStr: "", alignStr: "J", fill: false})

}

func CreateCoverPageRondo(pdf *gofpdf.Fpdf, firstPage *createpdffile.FirstPage) {
	pdf.AddPage()

	pageSize, pageHeight := pdf.GetPageSize()

	CreateRect(pdf, rondoLightBeige, RectInfo{0, 90, pageSize, pageHeight - 90, "F"})

	SetImageIntoPDF(pdf, firstPage.GetCompanyBigImage(), 60.587, 15.493, 43.592, 15.049, "logoLDA", false)

	SetImageIntoPDF(pdf, firstPage.GetCompanySmall(), 21.76, 10.442, 22.451, 24.072, "logoRondo", false)

	SetImageIntoPDF(pdf, firstPage.GetAppendix(), pageSize-63.875, 0, 63.875, pageHeight, "appendix", false)

	SetImageIntoPDF(pdf, firstPage.GetMainPageImage(), 71.929, 118.59, 225.071, 95.25, "mainPageLogo", false)

	// Картинки, которые я буду использовать в AddWatermark легче было загрузить сразу, а потом только использовать по их имени.
	// Именно поэтому они и 1 и 1 и -10, -10
	//SetImageIntoPDF(pdf, firstPage.GetSmallCoCompanyImage(), -10, -10, 1, 1, "leftImageIntoWaterMark", false)
	//SetImageIntoPDF(pdf, firstPage.GetCompanySmall(), -10, -10, 1, 1, "rightImageIntoWaterMark", false)
	//// Картинка, которая нужна при описании оборудования
	//// Именно поэтому она и 1 и 1 и -10, -10

	// Заголовок
	Text(
		pdf,
		greenColor,
		Font{font: "montserrat", style: "B", size: 43},
		Position{X: -1, Y: 47},
		MultiCellString{w: 0, h: 38 * math, txtStr: "Коммерческое", borderStr: "", alignStr: "L", fill: false},
	)
	Text(
		pdf,
		rondoBlack,
		Font{font: "montserrat", style: "B", size: 43},
		Position{X: -1, Y: 56},
		MultiCellString{w: 0, h: 38 * math, txtStr: "предложение", borderStr: "", alignStr: "L", fill: false},
	)

	// Исх № От Дата
	textStr := fmt.Sprintf("Исх. № %d от %s", firstPage.GetId(), time.Now().Format("02.01.2006"))
	Text(
		pdf,
		rondoBlack,
		Font{font: "montserrat", style: "M", size: 9},
		Position{X: 100, Y: 42.424},
		MultiCellString{w: 0, h: 10.8 * math, txtStr: textStr, borderStr: "", alignStr: "L", fill: false},
	)
	CreateRect(pdf, greenColor, RectInfo{101, 46, 48, 1, "F"})

	// Информация о человеке которому скидываем ПДФ
	Text(
		pdf,
		color000,
		Font{font: "inter", style: "B", size: 12},
		Position{X: -1, Y: 104},
		MultiCellString{w: 0, h: 14.4 * math, txtStr: firstPage.GetContactuser().GetFullName(), borderStr: "", alignStr: "L", fill: false},
	)

	fontInterBold12 := Font{font: "inter", style: "B", size: 12}
	fontInterRegular12 := Font{font: "inter", style: "", size: 12}
	fontInterItalic10 := Font{font: "inter", style: "I", size: 10}

	//pdf.SetFont("inter", "I", 10)
	//pdf.SetTextColor(0, 0, 0)
	//drawContactInfo(firstPage.GetContactuser())
	Text(pdf, color000, fontInterItalic10, Position{X: -1, Y: -1}, MultiCellString{0, 12 * math, drawContactInfo(firstPage.GetContactuser()), "", "L", false})
	// Объект
	TextCellFormat(pdf, color000, fontInterBold12, Position{X: 99.451, Y: 101.688}, CellString{0, 14.4 * math, "Объект:", "", 1, "L", false, 0, ""})
	TextCellFormat(pdf, color000, fontInterRegular12, Position{X: 99.451, Y: 105.688}, CellString{0, 14.4 * math, firstPage.GetObject(), "", 1, "L", false, 0, ""})

	// Кто выполнил (Менеджер)
	TextCellFormat(pdf, color000, fontInterBold12, Position{X: -1, Y: 120.779}, CellString{0, 14.4 * math, "Выполнил:", "", 1, "L", false, 0, ""})
	TextCellFormat(pdf, color000, fontInterRegular12, Position{X: -1, Y: 124.779}, CellString{0, 14.4 * math, firstPage.GetWhocreate().GetFullName(), "", 1, "L", false, 0, ""})
	//TextCellFormat(pdf, color000, fontInterItalic10, Position{X: -1, Y: -1}, CellString{0, 12 * math, firstPage.GetWhocreate().GetOccupy(), "", 1, "L", false, 0, ""})

	// Контактная информация
	TextCellFormat(pdf, greenColor, fontInterBold12, Position{X: -1, Y: 148.625}, CellString{0, 14.4 * math, "Контакты:", "", 1, "L", false, 0, ""})

	contactText := firstPage.GetContacts().GetPhone() + "\n" + firstPage.GetContacts().GetEmail() + "\n" + site
	Text(pdf, color000, fontInterRegular12, Position{X: -1, Y: 153.625}, MultiCellString{0, 14.4 * math, contactText, "", "L", false})
}

func CreateRect(pdf *gofpdf.Fpdf, color RGBColor, rectInfo RectInfo) {
	pdf.SetFillColor(color.R, color.G, color.B)
	pdf.Rect(rectInfo.x, rectInfo.y, rectInfo.w, rectInfo.h, rectInfo.styleStr)
}

func CreateRoundedRect(pdf *gofpdf.Fpdf, color RGBColor, roundedRectInfo RoundedRectInfo) {
	pdf.SetFillColor(color.R, color.G, color.B)
	pdf.RoundedRect(roundedRectInfo.x, roundedRectInfo.y, roundedRectInfo.w, roundedRectInfo.h, roundedRectInfo.r, roundedRectInfo.corners, roundedRectInfo.styleStr)
}

func CreateSystemFeaturesRondo(pdf *gofpdf.Fpdf, features []*createpdffile.SystemFeatures, number int) {
	CreateCircle(pdf, Position{28.568, 25.658}, 8.415, greenColor)
	//20.153, 16.243
	TextCellFormat(
		pdf,
		rondoWhite,
		Font{"montserrat", "B", 34},
		Position{25.428, 25.3},
		CellString{1, 1, strconv.Itoa(number), "", 0, "L", false, 0, ""},
	)
	//pdf.SetXY(25, 0)
	//pdf.SetFont("montserrat", "", 74)
	//pdf.SetTextColor(255, 89, 3)
	//pdf.CellFormat(50, 50, strconv.Itoa(number), "", 0, "L", false, 0, "")

	// Заголовок
	Text(
		pdf,
		rondoBlack,
		Font{"montserrat", "B", 28},
		Position{44.929, 17.355},
		MultiCellString{200, 10, "Особенности системы \nи требования заказчика", "", "L", false},
	)
	//pdf.SetFont("montserrat", "", 28)
	//pdf.SetTextColor(17, 22, 25)
	//pdf.SetXY(50, 13.5)
	//pdf.MultiCell(200, 10, "Особенности системы \nи требования заказчика", "", "L", false)

	items := make([]string, 0, len(features))
	for _, f := range features {
		items = append(items, f.GetFeature())
	}

	if pdf.GetImageInfo("appendix") != nil {
		pdf.Image("appendix", 156.45, 112.055, 140.55, 61.8, false, "", 0, "")
	}

	DrawNumberedListWordLikeRondo(pdf, items)

	AddWatermarkRondo(pdf)
}

func DrawNumberedListWordLikeRondo(pdf *gofpdf.Fpdf, items []string) {
	pdf.SetAutoPageBreak(false, 0)

	pageW, pageH := pdf.GetPageSize()

	const (
		startNumX = 20.718
		startY    = 55.449

		sidePad = 20.317
		colGap  = 8.0

		lineH      = 7.0
		paraSpace  = 3.0
		numW       = 8.0
		numTextGap = 3.0

		leftColHeight = 190.0
	)

	usableW := pageW - 2*sidePad
	colW := (usableW - colGap) / 2

	colX1 := startNumX
	colX2 := colX1 + colW + colGap

	bottomY1 := startY + leftColHeight
	bottomY2 := pageH - 80.0

	textW := colW - (numW + numTextGap)

	curCol := 0
	x0 := colX1
	y := startY

	baseFont := Font{"montserrat", "", 12}

	pdf.SetFont(baseFont.font, baseFont.style, baseFont.size)

	for i, text := range items {
		pdf.SetFont(baseFont.font, baseFont.style, baseFont.size)

		lines := pdf.SplitLines([]byte(text), textW)
		needH := float64(len(lines))*lineH + paraSpace

		bottomY := bottomY1
		if curCol == 1 {
			bottomY = bottomY2
		}

		if y+needH > bottomY {
			if curCol == 0 {
				curCol = 1
				x0 = colX2
				y = startY
				AddWatermarkRondo(pdf)
			} else {
				pdf.AddPage()
				pdf.SetAutoPageBreak(false, 0)
				pdf.SetFont(baseFont.font, baseFont.style, baseFont.size)

				if pdf.GetImageInfo("appendix") != nil {
					pdf.Image("appendix", 156.45, 112.055, 140.55, 61.8, false, "", 0, "")
				}

				curCol = 0
				x0 = colX1
				y = startY
			}
		}

		TextCellFormat(
			pdf, rondoBlack, baseFont,
			Position{x0, y},
			CellString{numW, lineH, strconv.Itoa(i+1) + ".", "", 0, "", false, 0, ""},
		)

		Text(
			pdf, rondoBlack, baseFont,
			Position{x0 + numW + numTextGap, y},
			MultiCellString{textW, lineH, text, "", "", false},
		)

		pdf.SetXY(x0, pdf.GetY())
		y = pdf.GetY() + paraSpace
	}
}

func DrawNumberedListWordLike(pdf *gofpdf.Fpdf, items []string, leftPad, rightPad, startY float64) {
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

func CreateTableOfContentsRondo(pdf *gofpdf.Fpdf, tableStartPage, characteristicsPage int, links map[string]LinkItem, order []string) {
	// Устанавливаем отступы
	//pdf.SetLeftMargin(78)
	pdf.SetLeftMargin(20.153)
	//pdf.SetTopMargin(44)
	pdf.SetY(16.243)

	// Заголовок
	TextCellFormat(
		pdf,
		rondoBlack,
		Font{"montserrat", "M", 24},
		Position{16.243, -1},
		CellString{0, 24 * math, "Содержание", "", 0, "L", false, 0, ""},
	)

	// Шрифт для содержания
	pdf.SetFont("inter", "", 12)
	pdf.SetY(36.127)

	// Функция для добавления пункта содержания
	addTOCItem := func(text string, pageNum int, link int, main bool) {
		y := pdf.GetY()

		// Смещение и оформление текста
		startX := 20.153
		if main {
			pdf.SetTextColor(greenColor.R, greenColor.G, greenColor.B)
			//text = strings.ToUpper(text)
		} else {
			pdf.SetTextColor(0, 0, 0)
			text = "     " + text // 5 пробелов
			startX += 3.656       // визуальное смещение
		}

		pdf.SetX(startX)

		// Рисуем основной текст
		if link > 0 {
			pdf.CellFormat(0, 16*math, text, "", 0, "L", false, link, "")
		} else {
			pdf.CellFormat(0, 16*math, text, "", 0, "L", false, 0, "")
		}

		// Ширины
		textWidth := pdf.GetStringWidth(text)
		pageWidth, _ := pdf.GetPageSize()

		dotStartX := startX + textWidth + 5

		// Номер страницы
		if pageNum > 0 {
			pageNumStr := strconv.Itoa(pageNum)
			pageNumWidth := pdf.GetStringWidth(pageNumStr)

			// было: pageNumX := pageWidth - 32 - 65
			pageNumX := pageWidth - 32 - 65 + 20 // сдвиг номера на 20mm вправо

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

	if pdf.GetImageInfo("logoRondo") != nil {
		pdf.Image("logoRondo", 40.539, 132.527, 23.073, 24.801, false, "", 0, "")
	}

	if pdf.GetImageInfo("logoLDA") != nil {
		pdf.Image("logoLDA", 80.399, 137.031, 46.099, 16.818, false, "", 0, "")
	}

	pageWidth, _ := pdf.GetPageSize()

	CreateRect(pdf, rondoLightBeige, RectInfo{0, 163.291, pageWidth, 46.711, "F"})
	CreateRoundedRect(pdf, rondoLightBeige, RoundedRectInfo{0, 24.327, 5, 41.416, 3.0, "23", "F"})

	if links["characteristics"].Page == 0 {
		AddWatermarkRondo(pdf)
	}
}

func CreateCircle(pdf *gofpdf.Fpdf, position Position, radius float64, color RGBColor) {
	pdf.SetFillColor(color.R, color.G, color.B)
	pdf.Circle(position.X, position.Y, radius, "F")
}

func ImageEquipmentsRondo(pdf *gofpdf.Fpdf, picture []byte, name, nameImage string, number, subNumber string) {
	runes := []rune(name)
	var result string
	if len(runes) > 4 {
		result = string(runes[4:])
	}

	CreateCircle(pdf, Position{28.568, 25.658}, 8.415, greenColor)

	TextCellFormat(
		pdf,
		rondoWhite,
		Font{"montserrat", "B", 26},
		Position{21.428, 25.3},
		CellString{1, 1, number + "." + subNumber, "", 0, "L", false, 0, ""},
	)

	Text(
		pdf,
		rondoBlack,
		Font{"montserrat", "B", 24},
		Position{44.929, 19.273},
		MultiCellString{200, 10, result, "", "L", false},
	)

	SetImageIntoPDF(pdf, picture, 26.84, 42.407, 243.346, 137.379, nameImage, true)

	AddWatermarkRondo(pdf)
}

func SelectsEquipmentsRondo(pdf *gofpdf.Fpdf, text string, number int) {
	const (
		leftX         = 26.612
		leftRightEdge = 150.0

		rightX      = 156.902
		rightMargin = 26.612

		leftStartY  = 40.0
		leftEndY    = 180.0
		rightStartY = 40.0
		rightEndY   = 130.0

		lineH  = 6 // GothamPro 12
		titleY = 20.178
		titleX = 44.929
	)

	CreateCircle(pdf, Position{28.568, 25.658}, 8.415, greenColor)

	TextCellFormat(
		pdf,
		rondoWhite,
		Font{"montserrat", "B", 34},
		Position{25.428, 25.3},
		CellString{1, 1, strconv.Itoa(number), "", 0, "L", false, 0, ""},
	)

	Text(
		pdf,
		rondoBlack,
		Font{"montserrat", "B", 28},
		Position{titleX, titleY},
		MultiCellString{200, 10, "Выбор оборудования", "", "L", false},
	)

	pageW, _ := pdf.GetPageSize()

	leftW := leftRightEdge - leftX
	rightW := (pageW - rightMargin) - rightX
	if leftW <= 0 {
		leftW = 80
	}
	if rightW <= 0 {
		rightW = 80
	}

	leftMaxLines := (leftEndY - leftStartY) / lineH
	rightMaxLines := (rightEndY - rightStartY) / lineH
	if leftMaxLines < 1 {
		leftMaxLines = 1
	}
	if rightMaxLines < 1 {
		rightMaxLines = 1
	}

	// Текстовый стиль
	pdf.SetFont("montserrat", "", 12)
	pdf.SetTextColor(rondoBlack.R, rondoBlack.G, rondoBlack.B)

	cur := newTextCursor(text)

	// Рисуем страницы пока курсор не закончится
	for {
		// Левая колонка
		leftLines := cur.fillLines(pdf, leftW, int(leftMaxLines))
		drawLinesColumn(pdf, leftX, leftStartY, leftW, lineH, leftLines, true)

		// Правая колонка
		rightLines := cur.fillLines(pdf, rightW, int(rightMaxLines))
		drawLinesColumn(pdf, rightX, rightStartY, rightW, lineH, rightLines, true)

		if pdf.GetImageInfo("appendix") != nil {
			pdf.Image("appendix", 173, 134.739, 88.35, 43.468, false, "", 0, "")
		}

		AddWatermarkRondo(pdf)

		if cur.done() {
			return
		}

		pdf.AddPage()
	}
}

func newTextCursor(text string) *textCursor {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return &textCursor{paras: nil, p: 0, w: 0}
	}

	// Важно: сохраняем пустые строки как отдельные "пустые параграфы"
	rawLines := strings.Split(text, "\n")
	paras := make([][]string, 0, len(rawLines))

	for _, ln := range rawLines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			paras = append(paras, nil) // пустая строка
			continue
		}
		paras = append(paras, strings.Fields(ln))
	}

	// Убираем хвостовые пустые строки, чтобы не плодить пустые страницы
	for len(paras) > 0 && len(paras[len(paras)-1]) == 0 {
		paras = paras[:len(paras)-1]
	}

	return &textCursor{paras: paras}
}

func (c *textCursor) done() bool {
	return c.p >= len(c.paras)
}

func (c *textCursor) fillLines(pdf *gofpdf.Fpdf, width float64, maxLines int) []string {
	if maxLines <= 0 || c.done() {
		return nil
	}

	lines := make([]string, 0, maxLines)

	for len(lines) < maxLines && !c.done() {
		para := c.paras[c.p]

		// Пустая строка/абзац
		if len(para) == 0 {
			lines = append(lines, "")
			c.p++
			c.w = 0
			continue
		}

		// Если параграф закончился — идём дальше
		if c.w >= len(para) {
			c.p++
			c.w = 0
			continue
		}

		// Собираем одну строку из слов по ширине
		word := para[c.w]

		// Слишком длинное слово — режем по ширине на куски
		if pdf.GetStringWidth(word) > width {
			chunks := splitLongWordByWidth(pdf, word, width)
			// Берем первый кусок как строку
			lines = append(lines, chunks[0])

			// Остаток кусков возвращаем обратно в поток слов
			restChunks := chunks[1:]
			if len(restChunks) > 0 {
				// заменяем текущее слово на restChunks[0], а остальное вставляем следом
				newPara := make([]string, 0, len(para)-c.w+len(restChunks))
				newPara = append(newPara, para[:c.w]...)
				newPara = append(newPara, restChunks...)
				newPara = append(newPara, para[c.w+1:]...)
				c.paras[c.p] = newPara
				// остаёмся на том же c.w (теперь там следующий chunk)
			} else {
				// слово полностью съедено
				c.w++
			}
			continue
		}

		cur := word
		c.w++

		for c.w < len(para) {
			next := para[c.w]
			try := cur + " " + next
			if pdf.GetStringWidth(try) <= width {
				cur = try
				c.w++
				continue
			}
			break
		}

		lines = append(lines, cur)
	}

	return lines
}

func drawLinesColumn(pdf *gofpdf.Fpdf, x, startY, w, lineH float64, lines []string, justify bool) {
	y := startY
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			y += lineH
			continue
		}

		out := line
		if justify && shouldJustifyLine(lines, i) {
			out = justifyToWidth(pdf, line, w)
		}

		pdf.SetXY(x, y)
		pdf.CellFormat(w, lineH, out, "", 0, "L", false, 0, "")
		y += lineH
	}
}

func shouldJustifyLine(lines []string, i int) bool {
	if i >= len(lines)-1 {
		return false
	}
	if lines[i] == "" || lines[i+1] == "" {
		return false
	}
	return strings.Contains(lines[i], " ")
}

func justifyToWidth(pdf *gofpdf.Fpdf, line string, targetW float64) string {
	words := strings.Fields(line)
	if len(words) < 2 {
		return line
	}

	spaceW := pdf.GetStringWidth(" ")
	if spaceW <= 0 {
		return line
	}

	wordsW := 0.0
	for _, w := range words {
		wordsW += pdf.GetStringWidth(w)
	}

	gaps := len(words) - 1
	curW := wordsW + float64(gaps)*spaceW

	extra := targetW - curW
	if extra <= spaceW {
		return line
	}

	addSpacesTotal := int(extra / spaceW)
	if addSpacesTotal <= 0 {
		return line
	}

	addPerGap := make([]int, gaps)
	for i := 0; i < addSpacesTotal; i++ {
		addPerGap[i%gaps]++
	}

	var b strings.Builder
	for i := 0; i < len(words); i++ {
		b.WriteString(words[i])
		if i < gaps {
			b.WriteString(" ")
			if addPerGap[i] > 0 {
				b.WriteString(strings.Repeat(" ", addPerGap[i]))
			}
		}
	}
	return b.String()
}

func splitLongWordByWidth(pdf *gofpdf.Fpdf, word string, maxW float64) []string {
	rs := []rune(word)
	if len(rs) == 0 {
		return []string{""}
	}

	var out []string
	start := 0
	for start < len(rs) {
		end := start + 1
		for end <= len(rs) && pdf.GetStringWidth(string(rs[start:end])) <= maxW {
			end++
		}
		end--
		if end <= start {
			end = start + 1
		}
		out = append(out, string(rs[start:end]))
		start = end
	}
	return out
}

func CreateModelTabel(pdf *gofpdf.Fpdf, pictures []*createpdffile.Models, amount *createpdffile.Amount, number, idCompany int) {
	const (
		maxTotalModelsFirstPageArstel = 3
		maxTotalModelsPageArstel      = 4
		maxTotalModelsFirstPageRondo  = 10
		maxTotalModelsPageRondo       = 11
	)

	//modelsOnCurrentPage := 0
	//tablePageNumber := 1
	//totalModels := len(pictures)
	//icon := pictures[0].Icon
	//needRub := false
	//var needText string
	//switch icon {
	//case "₽":
	//	needRub = true
	//	break
	//}

	// Настройка PDF
	if idCompany == 2 {
		pdf.SetLeftMargin(32)
		pdf.SetRightMargin(32)
		pdf.SetTopMargin(15)
	}

	// ПЕРВАЯ СТРАНИЦА
	var (
		headers       []string
		headerHeights float64
		needFull      bool
	)
	setSpecificationEquipment(pdf, idCompany, number)

	rows := make([][]Row, len(pictures))

	var wg sync.WaitGroup
	wg.Add(len(pictures))

	if idCompany == 2 {
		pdf.SetY(50)
		AddWatermark(pdf)

		headers = []string{
			"№",
			"Наименование,\nописание оборудования",
			"Кол-во,\nшт.",
			"Цена",
			"Сумма",
			"Наличие",
			"Фото",
		}

		headerHeights = 16.932
		//rowHeights = 36.686
		needFull = true
		//leftMargin = 0

		for i, r := range pictures {
			i, r := i, r
			go func() {
				defer wg.Done()

				rows[i] = []Row{
					{Width: tableWidthRecomendation[0], Text: strconv.Itoa(i + 1)},
					{Width: tableWidthRecomendation[1], Text: r.GetName()},
					{Width: tableWidthRecomendation[2], Text: strconv.Itoa(int(r.GetCount()))},
					{Width: tableWidthRecomendation[3], Text: r.GetDescription()},
				}
			}()
		}
	} else {
		headers = []string{
			"№",
			"Наименование оборудования и характеристики",
			"Кол-во, шт.",
			"Цена, $",
			"Сумма, $",
			"Наличие",
		}
		tableWidths = []float64{
			14,
			117.012,
			28.326,
			26.473,
			25.244,
			26.976,
		}
		headerHeights = 10
		needFull = false

		//leftMargin = 20.25
		AddWatermarkRondo(pdf)

		//rowHeights = 12.4
		/**
		message Models {
		  string name = 1;
		  string short_note = 2;
		  uint32 count = 3;
		  string presence = 4;
		  bytes img = 5;
		  float money_one = 6;
		  float money_count = 7;
		  string icon = 8;
		  string description = 9;
		  repeated ModelSpecials specials = 10;
		  FullInformation fullInformation = 11;
		}
		*/
		for i, r := range pictures {
			i, r := i, r
			go func() {
				defer wg.Done()

				rows[i] = []Row{
					{Width: tableWidths[0], Text: strconv.Itoa(i + 1)},
					{Width: tableWidths[1], Text: r.GetName()},
					//{Width: tableWidths[1], Text: r.GetShortNote()},
					{Width: tableWidths[2], Text: strconv.Itoa(int(r.GetCount()))},
					{Width: tableWidths[3], Text: strconv.Itoa(int(r.GetMoneyOne()))},
					{Width: tableWidths[4], Text: strconv.Itoa(int(r.GetMoneyCount()))},
					{Width: tableWidths[5], Text: r.GetPresence()},
				}
			}()
		}
	}

	wg.Wait()

	DrawTableHeader(pdf, tableWidths, headers, headerHeights, needFull, Font{"montserrat", "", 12}, RGBColor{140, 121, 104})

	//pdf.SetDrawColor(180, 180, 180)
	//pdf.SetLineWidth(0.3)
	//
	//// Обработка первой страницы (максимум 3 модели)
	//firstPageLimit := 3
	//modelsDrawn := 0
	//
	//for i := 0; i < totalModels && i < firstPageLimit; i++ {
	//	rowColor := getRowColor(i)
	//	pdf.SetFillColor(rowColor.R, rowColor.G, rowColor.B)
	//	drawTableRow(pdf, i, pictures[i], tableWidths, i%2 == 0)
	//	if pictures[i].GetPresence() == "Заказ" {
	//		needText = "По условиям договора поставка осуществляется при 100% предоплате со склада в Санкт-Петербурге.\nЦены указаны с учетом НДС 22%. Срок поставки оборудования под заказ  –  3 месяца с момента оплаты счета."
	//	}
	//	modelsDrawn++
	//	modelsOnCurrentPage++
	//}
	//
	//// Если на первой странице есть место для итогов (моделей < 3)
	//if totalModels < 3 {
	//	// Итоги на первой странице
	//	createSumm(pdf, amount, modelsDrawn%2 == 0, needRub, needText)
	//}
	//
	//// Если ровно 3 модели - новая страница для итогов
	//if totalModels == 3 {
	//	pdf.AddPage()
	//	tablePageNumber++
	//	// Только шапка на странице с итогами
	//	AddWatermark(pdf)
	//	pdf.SetY(20)
	//	drawTableHeaderForLandscape(pdf, tableWidths, 14.11)
	//	createSumm(pdf, amount, true, needRub, needText)
	//}
	//
	//// Если больше 3 моделей
	//modelsOnCurrentPage = 0
	//
	//// ВТОРАЯ И ПОСЛЕДУЮЩИЕ СТРАНИЦЫ
	//for i := 3; i < totalModels; i++ {
	//	// Если это начало новой страницы
	//	if modelsOnCurrentPage == 0 {
	//		pdf.AddPage()
	//		tablePageNumber++
	//		pdf.SetY(17)
	//		drawTableHeaderForLandscape(pdf, tableWidths, 14.11)
	//		AddWatermark(pdf)
	//	}
	//
	//	rowColor := getRowColor(modelsOnCurrentPage)
	//	pdf.SetFillColor(rowColor.R, rowColor.G, rowColor.B)
	//	drawTableRow(pdf, i, pictures[i], tableWidths, modelsOnCurrentPage%2 == 0)
	//	if pictures[i].GetPresence() == "Заказ" {
	//		needText = "По условиям договора поставка осуществляется при 100% предоплате со склада в Санкт-Петербурге.\nЦены указаны с учетом НДС 20%. Срок поставки оборудования под заказ  –  3 месяца с момента оплаты счета."
	//	}
	//	modelsOnCurrentPage++
	//	modelsDrawn++
	//
	//	// Проверяем лимит в 4 строки на странице
	//	if modelsOnCurrentPage >= 4 {
	//		// Если это последняя модель и страница полная - итоги на следующей
	//		if i == totalModels-1 {
	//			pdf.AddPage()
	//			tablePageNumber++
	//			pdf.SetY(20)
	//			drawTableHeaderForLandscape(pdf, tableWidths, 14.11)
	//			AddWatermark(pdf)
	//		}
	//		modelsOnCurrentPage = 0
	//	}
	//}
	//
	//createSumm(pdf, amount, modelsDrawn%2 == 1, needRub, needText)
}
