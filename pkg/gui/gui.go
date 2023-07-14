package gui

import (
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cfg = config.New()
var mu sync.Mutex = sync.Mutex{}

const limitLogs = 20
const limitConn = 30

type Data struct {
	Connections int
	NodesTotal  uint
}

type GUI struct {
	connections    []float64
	maxConnections int
	logs           []string
}

func New() *GUI {
	return &GUI{
		maxConnections: cfg.ConnectionsLimit,
	}
}

// TODO: optimize
func (g *GUI) Update(d Data) {
	mu.Lock()
	g.connections = append(g.connections, float64(d.Connections))
	if len(g.connections) > limitConn {
		g.connections = g.connections[len(g.connections)-limitConn:]
	}
	mu.Unlock()
}

// TODO: optimize
func (g *GUI) Log(log string) {
	mu.Lock()
	g.logs = append(g.logs, log)
	if len(g.logs) > limitLogs {
		g.logs = g.logs[len(g.logs)-limitLogs:]
	}
	mu.Unlock()
}

func (g *GUI) Start() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// fake data
	sinFloat64A := (func() []float64 {
		n := 400
		data := make([]float64, n)
		for i := range data {
			data[i] = 1 + math.Sin(float64(i)/5)
		}
		return data
	})()

	sinFloat64B := (func() []float64 {
		n := 400
		data := make([]float64, n)
		for i := range data {
			data[i] = 1 + math.Cos(float64(i)/5)
		}
		return data
	})()

	// PROGRESS
	g0 := widgets.NewGauge()
	g0.Title = "Progress"
	g0.Percent = 20
	g0.BarColor = ui.ColorGreen
	g0.BorderStyle.Fg = ui.ColorWhite
	g0.Label = fmt.Sprintf("%d/%d", 20, 100)
	g0.LabelStyle = ui.NewStyle(ui.ColorWhite)

	// CONNECTIONS
	sl := widgets.NewSparkline()
	// max connections
	sl.MaxVal = float64(g.maxConnections)
	sl.Data = sinFloat64A[:100]
	sl.LineColor = ui.ColorMagenta
	sl.TitleStyle.Fg = ui.ColorWhite
	slg := widgets.NewSparklineGroup(sl)
	slg.Title = "Connections"

	// STATS
	t1 := widgets.NewTable()
	t1.RowSeparator = false
	t1.FillRow = true
	t1.RowStyles[1] = ui.NewStyle(ui.ColorGreen)
	t1.RowStyles[2] = ui.NewStyle(ui.ColorRed)
	t1.RowStyles[3] = ui.NewStyle(ui.ColorYellow)
	t1.RowStyles[4] = ui.NewStyle(ui.ColorMagenta)
	t1.Rows = [][]string{
		[]string{"Total nodes", "10000"},
		[]string{"Good nodes", "123"},
		[]string{"Dead nodes", "456"},
		[]string{"Wait list", "789"},
		[]string{"Connections", fmt.Sprintf("%d/%d", 0, g.maxConnections)},
		[]string{"Msg out", "123"},
		[]string{"Msg in", "50"},
	}
	t1.TextStyle = ui.NewStyle(ui.ColorWhite)
	ui.Render(t1)

	// QUEUE
	chartQueue := widgets.NewPlot()
	chartQueue.ShowAxes = false
	chartQueue.Title = "Nodes"
	chartQueue.Data = append(chartQueue.Data, sinFloat64A)
	chartQueue.Data = append(chartQueue.Data, sinFloat64B)
	chartQueue.Data = append(chartQueue.Data, sinFloat64B)
	chartQueue.LineColors[0] = ui.ColorGreen  // good
	chartQueue.LineColors[1] = ui.ColorRed    // dead
	chartQueue.LineColors[2] = ui.ColorYellow // wait list
	chartQueue.PlotType = widgets.LineChart
	chartQueue.PlotType = widgets.LineChart
	// chartQueue.PlotType = widgets.ScatterPlot

	gs := make([]*widgets.Gauge, 3)
	for i := range gs {
		gs[i] = widgets.NewGauge()
		gs[i].Percent = i * 10
		gs[i].BarColor = ui.ColorRed
	}

	// LOGS
	p := widgets.NewParagraph()
	p.WrapText = true
	p.Text = "Loading..."
	p.Title = "Logs"

	// construct the result grid
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		// conn + stats + nodes
		ui.NewRow(1.0/2-1.0/5,
			ui.NewCol(0.2, slg),        // conn
			ui.NewCol(0.3, t1),         // stats
			ui.NewCol(0.5, chartQueue), // nodes chart
		),
		// progress
		ui.NewRow(1.0/10,
			ui.NewCol(1, g0),
		),
		// logs
		ui.NewRow(1.0/2,
			ui.NewCol(1.0, p),
		),
	)
	ui.Render(grid)

	// UPDATER
	tickerCount := 1
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}
		case <-ticker:
			if tickerCount == 100 {
				return
			}
			for _, g := range gs {
				g.Percent = (g.Percent + 3) % 100
			}
			// update logs

			p.Text = strings.Join(g.logs, "\n")
			// connections update
			// slg.Sparklines[0].Data = sinFloat64B[tickerCount : tickerCount+100]
			slg.Sparklines[0].Data = g.connections
			// slg.Sparklines[1].Data = sinFloat64B[tickerCount : tickerCount+100]
			// chartConn.Data[0] = sinFloat64A[2*tickerCount:]
			// chartQueue.Data[0] = sinFloat64B[2*tickerCount:]
			chartQueue.Data[0] = sinFloat64A[2*tickerCount:]
			chartQueue.Data[1] = sinFloat64B[2*tickerCount:]
			chartQueue.Data[2] = sinFloat64B[1*tickerCount:]
			// chartConn.LineColors[0] = ui.ColorMagenta
			// chartQueue.LineColors[0] = ui.ColorYellow
			// lc.Data[1] = sinFloat64B[2*tickerCount:]
			ui.Render(grid)
			tickerCount++
		}
	}
}
