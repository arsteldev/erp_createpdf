package createPDF

import (
	"bytes"
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
	// Темно-серый фон #3d4a4d
	pdf.SetFillColor(61, 74, 77)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Inter", "B", 9.5)

	// Для шапки делаем границы того же цвета что и фон
	pdf.SetDrawColor(61, 74, 77)

	//headers := []string{
	//	"№",
	//	"Наименование,\nописание оборудования",
	//	"Кол-во,\nшт.",
	//	"Цена",
	//	"Сумма",
	//	"Наличие",
	//	"Фото",
	//}

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
