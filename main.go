package main

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type GameState int

const (
	StateMenu GameState = iota
	StateTyping
	StateResults
)

type Game struct {
	state       GameState
	targetText  string
	targetLines []string
	userText    string

	hover      bool
	hoverAlpha float64

	fontBig   font.Face
	fontSmall font.Face
}

func makeWordsSlice() []string {
	jsonBytes, err := os.ReadFile("scripts/words.json")
	if err != nil {
		panic(err)
	}

	data := make(map[string]int)
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		panic(err)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	n := 200
	randomKeys := keys[:n]
	return randomKeys
}

func makeStrings(s []string) []string {
	resultStrings := make([]string, 0, (len(s)/5)+1)

	for i := 0; i < len(s); {
		var str string
		for counter := 0; counter < 5; counter++ {
			str = str + s[i] + " "
			i++
		}
		resultStrings = append(resultStrings, str)
	}
	return resultStrings
}

var russianFont font.Face

// ===============================
// СЛОЙ ШРИФТА
// ===============================

func loadFont(path string, size float64) font.Face {
	fontData, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	tt, err := opentype.Parse(fontData)
	if err != nil {
		panic(err)
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
	return face
}

// ===============================
// UPDATE
// ===============================

func (g *Game) Update() error {

	switch g.state {

	// ---------------- MENU ----------------
	case StateMenu:
		mx, my := ebiten.CursorPosition()
		startX, startY := 340, 280
		startW, startH := 150, 50

		// Проверка наведения
		g.hover = mx >= startX && mx <= startX+startW &&
			my >= startY && my <= startY+startH

		// Плавный hover (0 → 1)
		if g.hover {
			g.hoverAlpha += 0.1
			if g.hoverAlpha > 1 {
				g.hoverAlpha = 1
			}
		} else {
			g.hoverAlpha -= 0.1
			if g.hoverAlpha < 0 {
				g.hoverAlpha = 0
			}
		}

		// Клик мышью
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.hover {
			g.startTyping()
		}

		// ENTER
		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			g.startTyping()
		}

	// ---------------- TYPING ----------------
	case StateTyping:

		// Здесь позже добавим обработку клавиш
		// Сейчас ESC возвращает в меню
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			g.state = StateMenu
		}

	// ---------------- RESULTS ----------------
	case StateResults:
		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			g.state = StateMenu
		}
	}

	return nil
}

// Запуск режима печати
func (g *Game) startTyping() {
	g.state = StateTyping
	g.userText = ""
}

// ===============================
// DRAW
// ===============================

func (g *Game) Draw(screen *ebiten.Image) {

	switch g.state {
	case StateMenu:
		g.drawMenu(screen)
	case StateTyping:
		g.drawTyping(screen)
	case StateResults:
		g.drawResults(screen)
	}
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	text.Draw(screen, "Тест скорости печати", g.fontBig, 275, 180, color.White)
	text.Draw(screen, "Нажмите кнопку или Enter для старта", g.fontSmall, 250, 230, color.White)

	startX, startY := 325, 280
	startW, startH := 150, 50

	// Цвет кнопки с плавной анимацией
	brightness := uint8(60 + int(40*g.hoverAlpha))
	btnColor := color.RGBA{brightness, brightness, brightness, 255}

	ebitenutil.DrawRect(screen, float64(startX), float64(startY), float64(startW), float64(startH), btnColor)
	text.Draw(screen, "СТАРТ", g.fontBig, startX+35, startY+33, color.White)
}

func (g *Game) drawTyping(screen *ebiten.Image) {
	text.Draw(screen, "Режим печати (ESC — выход)", g.fontSmall, 260, 80, color.White)

	text.Draw(screen, "Текст:", g.fontSmall, 100, 200, color.White)

	baseY := 240
	lineHeight := 25

	for i, line := range g.targetLines {
		y := baseY + i*lineHeight
		text.Draw(screen, line, g.fontBig, 100, y, color.White)
	}

	text.Draw(screen, "Ваш ввод:", g.fontSmall, 100, 340, color.White)
	text.Draw(screen, g.userText, g.fontBig, 100, 360, color.White)
}

func (g *Game) drawResults(screen *ebiten.Image) {
	text.Draw(screen, "Результаты (Enter — в меню)", g.fontBig, 200, 200, color.White)
}

// ===============================
// LAYOUT
// ===============================

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

// ===============================
// MAIN
// ===============================

func main() {
	s := makeWordsSlice()
	resStr := makeStrings(s)

	renderLines := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		renderLines = append(renderLines, resStr[i])
	}

	game := &Game{
		state:       StateMenu,
		targetText:  resStr[0],
		targetLines: renderLines,
	}

	// Загружаем шрифты
	game.fontBig = loadFont("font.ttf", 24)
	game.fontSmall = loadFont("font.ttf", 16)

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Typing Speed Test")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
