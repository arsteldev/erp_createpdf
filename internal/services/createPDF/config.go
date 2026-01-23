package createPDF

type RGBColor struct {
	R, G, B int
}

type Font struct {
	font  string
	style string
	size  float64
}

type MultiCellString struct {
	w         float64
	h         float64
	txtStr    string
	borderStr string
	alignStr  string
	fill      bool
}

type CellString struct {
	w         float64
	h         float64
	txtStr    string
	borderStr string
	ln        int
	alignStr  string
	fill      bool
	link      int
	linkStr   string
}

type Position struct {
	X, Y float64
}

type Parametrs struct {
	Width, Height float64
}

type Row struct {
	Width float64
	Text  string
}
