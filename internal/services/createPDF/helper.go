package createPDF

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/jung-kurt/gofpdf"
	"image/png"
	"strings"
)

func Text(pdf *gofpdf.Fpdf, rgb RGBColor, font Font, position Position, cellString MultiCellString) {
	pdf.SetFont(font.font, font.style, font.size)
	pdf.SetTextColor(rgb.R, rgb.G, rgb.B)
	if position.X == -1 && position.Y != -1 {
		pdf.SetY(position.Y)
	} else if position.X != -1 && position.Y != -1 {
		pdf.SetXY(position.X, position.Y)
	}
	pdf.MultiCell(cellString.w, cellString.h, cellString.txtStr, cellString.borderStr, cellString.alignStr, cellString.fill)
}

func TextCellFormat(pdf *gofpdf.Fpdf, rgb RGBColor, font Font, position Position, cellString CellString) {
	if position.X == -1 && position.Y != -1 {
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

func DrawTableHeader(pdf *gofpdf.Fpdf, widths []float64, headers []string, headerHeights float64) {
	pdf.SetY(50)
	// Темно-серый фон #3d4a4d
	//pdf.SetFillColor(61, 74, 77)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Inter", "B", 9.5)

	// Для шапки делаем границы того же цвета что и фон
	pdf.SetDrawColor(61, 74, 77)

	currentY := pdf.GetY()
	startX := pdf.GetX()

	// 1. Сначала закрашиваем всю полосу от левого края до правого
	pageWidth, _ := pdf.GetPageSize()
	drawBackgroud(pdf, Position{X: 0, Y: currentY}, Parametrs{Width: pageWidth, Height: headerHeights}, RGBColor{R: 61, G: 74, B: 77})
	//leftMargin := 0.0
	//rightMargin := 0.0
	//
	//// Закрашиваем всю полосу
	//pdf.SetXY(leftMargin, currentY)
	//pdf.CellFormat(pageWidth-leftMargin-rightMargin, headerHeights, "", "0", 0, "C", true, 0, "")

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

func DrawTableRows(pdf *gofpdf.Fpdf, rows [][]Row, headerHeight float64) {
	const (
		firstPageRows = 3
		nextPageRows  = 4
		headerPad     = 0.5
	)

	firstPage := true
	rowsOnPage := 0

	// Старт под шапкой + небольшой отступ
	if pdf.GetY() < headerHeight+headerPad {
		pdf.SetY(headerHeight + headerPad)
	}

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
			pdf.SetY(((headerHeight * 2) - 1) + headerPad)
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
		drawRow(pdf, row, pos, bg)

		rowsOnPage++
	}
}

// drawRow — рисует одну строку таблицы (фон + ячейки)
func drawRow(pdf *gofpdf.Fpdf, columns []Row, position Position, color RGBColor) {
	// Ширина страницы
	pageWidth, _ := pdf.GetPageSize()

	// Фон строки
	drawBackgroud(pdf, Position{X: 0, Y: position.Y}, Parametrs{Width: pageWidth, Height: rowHeight}, color)

	// Текст по ячейкам
	x := position.X
	y := position.Y
	pdf.SetXY(x, y)

	for _, c := range columns {
		pdf.SetXY(x, y)
		pdf.SetTextColor(0, 0, 0)
		// ВАЖНО: используйте MultiCell, если возможны переносы по строкам
		pdf.CellFormat(c.Width, rowHeight, c.Text, "", 0, "L", false, 0, "")
		x += c.Width
	}

	// Переход на следующую строку
	pdf.SetXY(position.X, position.Y+rowHeight)
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
		siteX := pageNumX - 30
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
