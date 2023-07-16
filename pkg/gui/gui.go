package gui

import (
	"context"
	"fmt"
	"go-btc-downloader/pkg/config"
	buff "go-btc-downloader/pkg/gui/buffer"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	tui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cfg = config.New()
var mu sync.Mutex = sync.Mutex{}

const lenLogs = 20
const lenConnChart = 14
const lenNodesChart = 32

type IncomingData struct {
	Connections int
	NodesTotal  int
	NodesGood   int
	NodesDead   int
	NodesQueued int
	Log         string
}

type GUI struct {
	ctx             context.Context
	ch              chan IncomingData
	buffConnections *buff.GuiBuffer
	buffNodesTotal  *buff.GuiBuffer
	buffNodesQueued *buff.GuiBuffer
	buffNodesGood   *buff.GuiBuffer
	buffNodesDead   *buff.GuiBuffer
	buffLogs        *buff.GuiBuffer
}

func New(ctx context.Context, ch chan IncomingData) *GUI {
	g := GUI{
		ctx:             ctx,
		ch:              ch,
		buffConnections: buff.New(lenConnChart),
		buffNodesTotal:  buff.New(lenNodesChart),
		buffNodesQueued: buff.New(lenNodesChart),
		buffNodesGood:   buff.New(lenNodesChart),
		buffNodesDead:   buff.New(lenNodesChart),
		buffLogs:        buff.New(lenLogs),
	}
	return &g
}

func (g *GUI) listner() {
	for {
		select {
		case <-g.ctx.Done():
			return
		case d := <-g.ch:
			mu.Lock()
			g.buffConnections.AddNum(d.Connections)
			g.buffNodesTotal.AddNum(d.NodesTotal)
			g.buffNodesQueued.AddNum(d.NodesQueued)
			g.buffNodesGood.AddNum(d.NodesGood)
			g.buffNodesDead.AddNum(d.NodesDead)
			g.buffLogs.AddString(d.Log)
			mu.Unlock()
		}
	}
}

func (g *GUI) Stop() {
	tui.Close()
}

func (g *GUI) Start() {
	if err := tui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer func() {
		tui.Close()
		log.Println("ready to exit")
	}()

	// start incoming data listner
	go g.listner()

	// PROGRESS
	g0 := widgets.NewGauge()
	g0.Title = "Progress"
	g0.Percent = 20
	g0.BarColor = tui.ColorBlue
	g0.BorderStyle.Fg = tui.ColorWhite
	g0.Label = fmt.Sprintf("%d/%d", 20, 100)
	g0.LabelStyle = tui.NewStyle(tui.ColorWhite)

	// CONNECTIONS
	chartConn := widgets.NewSparkline()
	// max connections
	chartConn.MaxVal = float64(cfg.ConnectionsLimit) * 1.2 // height hack
	chartConn.Data = []float64{0}
	chartConn.LineColor = tui.ColorMagenta
	chartConn.TitleStyle.Fg = tui.ColorWhite
	chartConnWrap := widgets.NewSparklineGroup(chartConn)
	chartConnWrap.Title = "Connections"

	// STATS
	stats := widgets.NewTable()
	stats.RowSeparator = false
	stats.FillRow = false
	stats.RowStyles[1] = tui.NewStyle(tui.ColorGreen)
	stats.RowStyles[2] = tui.NewStyle(tui.ColorRed)
	stats.RowStyles[3] = tui.NewStyle(tui.ColorYellow)
	stats.RowStyles[4] = tui.NewStyle(tui.ColorMagenta)
	stats.Rows = g.getInfo()
	stats.TextStyle = tui.NewStyle(tui.ColorWhite)
	tui.Render(stats)

	// TOTAL
	chartNodesTotal := widgets.NewPlot()
	chartNodesTotal.ShowAxes = false
	chartNodesTotal.Data = [][]float64{make([]float64, lenNodesChart)}
	chartNodesTotal.LineColors = []tui.Color{tui.ColorWhite} // force the collor, bug

	// QUEUE
	chartNodesQueue := widgets.NewPlot()
	chartNodesQueue.ShowAxes = false
	chartNodesQueue.Data = [][]float64{make([]float64, lenNodesChart)}
	chartNodesQueue.LineColors = []tui.Color{tui.ColorYellow} // force the collor, bug

	// good
	chartNodesGood := widgets.NewPlot()
	chartNodesGood.ShowAxes = false
	chartNodesGood.Data = [][]float64{make([]float64, lenNodesChart)}
	chartNodesGood.LineColors = []tui.Color{tui.ColorGreen} // force the collor, bug

	// dead
	chartNodesDead := widgets.NewPlot()
	chartNodesDead.ShowAxes = false
	chartNodesDead.Data = [][]float64{make([]float64, lenNodesChart)}
	chartNodesDead.LineColors = []tui.Color{tui.ColorRed} // force the collor, bug

	gs := make([]*widgets.Gauge, 3)
	for i := range gs {
		gs[i] = widgets.NewGauge()
		gs[i].Percent = i * 10
		gs[i].BarColor = tui.ColorRed
	}

	// LOGS
	p := widgets.NewParagraph()
	p.WrapText = true
	p.Text = "Loading..."
	p.Title = "Logs"

	// construct the result grid
	grid := tui.NewGrid()
	termWidth, termHeight := tui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		// conn + stats + nodes
		tui.NewRow(0.25,
			tui.NewCol(0.2, stats),
			tui.NewCol(0.2, chartNodesTotal),
			tui.NewCol(0.2, chartNodesQueue),
			tui.NewCol(0.2, chartNodesGood),
			tui.NewCol(0.2, chartNodesDead),
		),
		// logs
		tui.NewRow(0.65,
			tui.NewCol(0.9, p),
			tui.NewCol(0.1, chartConnWrap),
		),
		// progress
		tui.NewRow(0.1,
			tui.NewCol(1, g0),
		),
	)
	tui.Render(grid)

	// send debug data
	if os.Getenv("GUI_DEBUG") == "1" {
		go g.sendDebugData()
	}

	// UPDATER
	tickerCount := 1
	uiEvents := tui.PollEvents()
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-g.ctx.Done():
			return
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(tui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				tui.Clear()
				tui.Render(grid)
			}
		case <-ticker.C:
			for _, g := range gs {
				g.Percent = (g.Percent + 3) % 100
			}

			// update logs
			p.Text = g.buffLogs.GetString()

			// connections update
			chartConnWrap.Sparklines[0].Data = g.buffConnections.GetFloats()

			// nodes chart
			// chartNodesTotal.Data[0] = g.dataNodesTotal.Data()
			chartNodesTotal.Data[0] = g.buffNodesTotal.GetFloats()
			chartNodesQueue.Data[0] = g.buffNodesQueued.GetFloats()
			chartNodesGood.Data[0] = g.buffNodesGood.GetFloats()
			chartNodesDead.Data[0] = g.buffNodesDead.GetFloats()

			//  update titles
			updateTitle(chartNodesTotal, g.buffNodesTotal.GetLastNum(), "Total")
			updateTitle(chartNodesQueue, g.buffNodesQueued.GetLastNum(), "Queue")
			updateTitle(chartNodesGood, g.buffNodesGood.GetLastNum(), "Good")
			updateTitle(chartNodesDead, g.buffNodesDead.GetLastNum(), "Dead")
			updateTitleChart(chartConnWrap, g.buffConnections.GetLastNum(), "Conn.")

			// update info
			stats.Rows = g.getInfo()

			// debug info to logs
			if os.Getenv("GUI_MEM") == "1" {
				msg := fmt.Sprintf("buffNodesTotal: len %d, cap %d\n", len(g.buffNodesTotal.GetFloats()), cap(g.buffNodesTotal.GetFloats()))
				msg += fmt.Sprintf("buffNodesQueued: len %d, cap %d\n", len(g.buffNodesQueued.GetFloats()), cap(g.buffNodesQueued.GetFloats()))
				msg += fmt.Sprintf("buffNodesGood: len %d, cap %d\n", len(g.buffNodesGood.GetFloats()), cap(g.buffNodesGood.GetFloats()))
				msg += fmt.Sprintf("buffNodesDead: len %d, cap %d\n", len(g.buffNodesDead.GetFloats()), cap(g.buffNodesDead.GetFloats()))
				msg += fmt.Sprintf("buffConnections: len %d, cap %d\n", len(g.buffConnections.GetFloats()), cap(g.buffConnections.GetFloats()))

				// msg += fmt.Sprintf("dataNodesTotalLL: %d\n", g.dataNodesTotalList.Len())
				// report G count and memory used
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				msg += fmt.Sprintf("STATS: G:%d, MEM:%dKb\n", runtime.NumGoroutine(), m.Alloc/1024)
				p.Text = msg
			}
			tui.Render(grid)
			tickerCount++
		}
	}
}

// update GUI data
// in serial data (charts) we push new data to the linked lists first
// and then construct slices from the linked lists
// func (g *GUI) Update(d IncomingData) {
// 	mu.Lock()
// 	g.buffConnections.AddNum(d.Connections)
// 	g.buffNodesTotal.AddNum(d.NodesTotal)
// 	g.buffNodesQueued.AddNum(d.NodesQueued)
// 	g.buffNodesGood.AddNum(d.NodesGood)
// 	g.buffNodesDead.AddNum(d.NodesDead)
// 	mu.Unlock()
// }

func (g *GUI) Log(log string) {
	g.buffLogs.AddString(log)
}

func (g *GUI) getInfo() [][]string {
	return [][]string{
		{"Total nodes", fmt.Sprintf("%d", g.buffNodesTotal.GetLastNum())},
		{"Good nodes", fmt.Sprintf("%d", g.buffNodesGood.GetLastNum())},
		{"Dead nodes", fmt.Sprintf("%d", g.buffNodesDead.GetLastNum())},
		{"Queue", fmt.Sprintf("%d", g.buffNodesQueued.GetLastNum())},
		{"Connections", fmt.Sprintf("%d/%d", g.buffConnections.GetLastNum(), cfg.ConnectionsLimit)},
	}
}

// update titles
func updateTitleChart(chart *widgets.SparklineGroup, data int, title string) {
	if data > 0 {
		title += fmt.Sprintf(": %d", data)
	}
	chart.Title = title
}

func updateTitle(chart *widgets.Plot, data int, title string) {
	if data > 0 {
		title += fmt.Sprintf(" (%d)", data)
	}
	chart.Title = title
}

func (g *GUI) sendDebugData() {
	// send fake data to gui for debug
	cnt := 0
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < 5; i++ {
				cnt = cnt + i
				g.ch <- IncomingData{Log: fmt.Sprintf("test log %d\n", cnt)}
			}
			rConn := rand.Intn(cfg.ConnectionsLimit)
			rTotal := rand.Intn(cfg.ConnectionsLimit)
			rQueued := rand.Intn(cfg.ConnectionsLimit)
			rGood := rand.Intn(cfg.ConnectionsLimit)
			rDead := rand.Intn(cfg.ConnectionsLimit)
			g.ch <- IncomingData{
				Connections: rConn,
				NodesTotal:  rTotal,
				NodesQueued: rQueued,
				NodesGood:   rGood,
				NodesDead:   rDead,
			}
		}
	}
}
