package main

import (
	"encoding/json"
	"fmt"
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
	StateStartTyping
	StateResults
)

type Game struct {
	state       GameState
	targetText  string
	targetLines []string
	userText    string
	currentPos  int
	counter     Counter
	myTimer     Timer

	allLines   []string
	lineOffset int

	correctText string

	currentResult Results

	hover      bool
	hoverAlpha float64

	fontBig   font.Face
	fontSmall font.Face
}

type Counter struct {
	allSignCounter   int
	rightSignCounter int
	accuracy         float64
}

func (c *Counter) incrementAllSignCounter() {
	c.allSignCounter++
}

func (c *Counter) incrementRightSignCounter() {
	c.rightSignCounter++
}

func (c *Counter) calculateAccuracy() {
	if c.allSignCounter == 0 {
		c.accuracy = 0
		return
	}
	c.accuracy = (float64(c.rightSignCounter) / float64(c.allSignCounter)) * 100
}

type Timer struct {
	duration    time.Duration
	startTime   time.Time
	endTime     time.Time
	isRunning   bool
	remaining   time.Duration
	timeElapsed time.Duration
}

func NewTimer(duration time.Duration) *Timer {
	return &Timer{
		duration:  duration,
		remaining: duration,
	}
}

func (t *Timer) Start() {
	if t.isRunning {
		fmt.Println("Таймер уже запущен!")
		return
	}

	t.isRunning = true
	t.startTime = time.Now()
	t.endTime = t.startTime.Add(t.duration)
	t.remaining = t.duration
	t.timeElapsed = 0
}

func (t *Timer) Update() {
	if !t.isRunning {
		return
	}

	now := time.Now()
	t.timeElapsed = now.Sub(t.startTime)
	t.remaining = t.duration - t.timeElapsed

	if t.remaining <= 0 {
		t.remaining = 0
		t.isRunning = false
	}
}

func (t *Timer) GetRemainingFormatted() string {
	if t.remaining <= 0 {
		return "00:00"
	}

	totalSeconds := int(t.remaining.Round(time.Second).Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (t *Timer) GetTimeElapsed() time.Duration {
	return t.timeElapsed
}

func (t *Timer) Stop() {
	t.isRunning = false
}

func (t *Timer) IsFinished() bool {
	return t.remaining <= 0
}

func (t *Timer) IsRunning() bool {
	return t.isRunning
}

type Results struct {
	TypingSpeed int     `json:"typingSpeed"`
	Accuracy    float64 `json:"accuracy"`
	Timestamp   string  `json:"timestamp"`
}

// Функция для загрузки текущего лучшего результата
func loadBestResult(filename string) (Results, error) {
	var bestResult Results

	file, err := os.ReadFile(filename)
	if err != nil {
		// Если файла нет, возвращаем пустой результат
		if os.IsNotExist(err) {
			return Results{}, nil
		}
		return Results{}, err
	}

	err = json.Unmarshal(file, &bestResult)
	if err != nil {
		return Results{}, err
	}

	return bestResult, nil
}

// Простая версия - сравниваем по "очкам" (скорость × точность)
func saveIfBetterSimple(newResult Results, filename string) error {
	bestResult, err := loadBestResult(filename)
	if err != nil {
		return err
	}

	// Вычисляем "очки" для сравнения
	newScore := float64(newResult.TypingSpeed) * (newResult.Accuracy / 100)
	bestScore := float64(bestResult.TypingSpeed) * (bestResult.Accuracy / 100)

	// Если новый результат лучше ИЛИ это первый результат
	if newScore > bestScore || (bestResult.TypingSpeed == 0 && bestResult.Accuracy == 0) {
		data, err := json.MarshalIndent(newResult, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filename, data, 0644)
	}

	return nil
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

func (g *Game) drawBestResult() string {
	best, err := loadBestResult("best_result.json")
	if err != nil {
		return ""
	}

	if best.TypingSpeed > 0 {
		bestText := fmt.Sprintf("Лучший результат:\nСкорость печати: %d зн/мин\nТочность: %.2f%%",
			best.TypingSpeed, best.Accuracy)
		return bestText
	}
	return ""
}

func (g *Game) drawCurrentResult() string {
	results := fmt.Sprintf("Текущий результат:\nСкорость печати: %d зн/мин\nТочность: %.2f%%",
		g.currentResult.TypingSpeed,
		g.currentResult.Accuracy)

	return results
}

func (g *Game) resetGame() {
	// Просто пересоздаем все данные как в main()
	s := makeWordsSlice()
	g.allLines = makeStrings(s)
	g.lineOffset = 0
	g.currentPos = 0
	g.correctText = ""
	g.userText = ""
	g.counter = Counter{}
	g.currentResult = Results{}
	g.targetLines = g.allLines[g.lineOffset:min(g.lineOffset+3, len(g.allLines))]
	if len(g.targetLines) > 0 {
		g.targetText = g.targetLines[0]
	}
	g.myTimer = Timer{}
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

	// ---------------- TYPING ----------------
	case StateTyping:

		// Сейчас ESC возвращает в меню
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			g.state = StateMenu
		}

		// Обработка Enter -> StateStartTyping
		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			g.resetGame() // Сбрасываем игру перед началом таймера
			g.myTimer = *NewTimer(60 * time.Second)
			g.myTimer.Start()
			g.state = StateStartTyping
		}

	// ----------------Start_Typing ----------------
	case StateStartTyping:
		// Обновляем таймер
		g.myTimer.Update()

		if g.myTimer.IsFinished() {
			var accuracy float64
			if g.counter.allSignCounter > 0 {
				accuracy = (float64(g.counter.rightSignCounter) / float64(g.counter.allSignCounter)) * 100
			}

			g.currentResult = Results{
				TypingSpeed: g.counter.rightSignCounter,
				Accuracy:    accuracy,
				Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
			}

			// Сохраняем только если результат лучше
			if err := saveIfBetterSimple(g.currentResult, "best_result.json"); err != nil {
				fmt.Printf("Ошибка сохранения: %v\n", err)
			} else {
				// Можно показать сообщение, что это новый рекорд
				fmt.Println("Новый рекорд!")
			}

			g.state = StateResults
			break
		}

		inputRunes := ebiten.InputChars()

		if len(inputRunes) > 0 {
			targetRunes := []rune(g.targetText)

			if g.currentPos < len(targetRunes) {
				exceptedRune := targetRunes[g.currentPos]
				if inputRunes[0] == exceptedRune {
					g.counter.incrementRightSignCounter()
					g.correctText += string(exceptedRune)
					g.userText = string(inputRunes)
					g.currentPos++
				} else {
					g.userText = string(inputRunes)
				}
				g.counter.incrementAllSignCounter() // Всегда увеличиваем общий счетчик
			}

			if g.currentPos >= len(targetRunes) {
				g.correctText = ""
				g.lineOffset++

				g.targetLines = g.allLines[g.lineOffset:min(g.lineOffset+3, len(g.allLines))]

				if len(g.targetLines) > 0 {
					g.targetText = g.targetLines[0]
					g.currentPos = 0
				} else {
					g.state = StateResults
				}
			}
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
	case StateStartTyping:
		g.drawStartTyping(screen)
	case StateResults:
		g.drawResults(screen)
	}
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	text.Draw(screen, "Тест скорости печати", g.fontBig, 275, 180, color.White)

	startX, startY := 325, 280
	startW, startH := 150, 50

	// Цвет кнопки с плавной анимацией
	brightness := uint8(60 + int(40*g.hoverAlpha))
	btnColor := color.RGBA{brightness, brightness, brightness, 255}

	ebitenutil.DrawRect(screen, float64(startX), float64(startY), float64(startW), float64(startH), btnColor)
	text.Draw(screen, "СТАРТ", g.fontBig, startX+35, startY+33, color.White)

}

func (g *Game) drawTyping(screen *ebiten.Image) {
	text.Draw(screen, "Нажмите Enter чтобы начать", g.fontBig, 240, 300, color.White)
	text.Draw(screen, "ESC — выход", g.fontSmall, 350, 340, color.White)
	text.Draw(screen, "01:00", g.fontBig, 370, 500, color.White)

}

func (g *Game) drawStartTyping(screen *ebiten.Image) {
	text.Draw(screen, "Текст:", g.fontSmall, 100, 200, color.White)

	baseY := 240
	lineHeight := 25

	for i, line := range g.targetLines {
		y := baseY + i*lineHeight
		text.Draw(screen, line, g.fontBig, 100, y, color.White)
	}

	text.Draw(screen, "Ваш ввод:", g.fontSmall, 100, 340, color.White)
	text.Draw(screen, g.userText, g.fontBig, 100, 360, color.White)

	stats := fmt.Sprintf("Правильно: %d/%d", g.counter.rightSignCounter, g.counter.allSignCounter)
	text.Draw(screen, stats, g.fontSmall, 100, 400, color.White)

	text.Draw(screen, g.correctText, g.fontBig, 100, 240, color.RGBA{32, 235, 45, 255})

	// Отображение оставшегося времени
	timeText := fmt.Sprintf("%s", g.myTimer.GetRemainingFormatted())
	text.Draw(screen, timeText, g.fontBig, 370, 500, color.White)
}

func (g *Game) drawResults(screen *ebiten.Image) {
	text.Draw(screen, "Нажмите Enter для выхода в меню", g.fontSmall, 250, 500, color.White)

	// Вывод результата текущей попытки
	results := g.drawCurrentResult()
	text.Draw(screen, results, g.fontBig, 10, 240, color.White)

	// Вывод рекорда
	res := g.drawBestResult()
	text.Draw(screen, res, g.fontBig, 450, 240, color.White)
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

// Вспомогательная функция для безопасного среза
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	s := makeWordsSlice()
	allLines := makeStrings(s)

	game := &Game{
		state:       StateMenu,
		allLines:    allLines,
		lineOffset:  0,
		correctText: "",
	}

	game.targetLines = allLines[game.lineOffset:min(game.lineOffset+3, len(allLines))]
	if len(game.targetLines) > 0 {
		game.targetText = game.targetLines[0]
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
