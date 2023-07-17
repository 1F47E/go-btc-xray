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

const LEN_LOGS = 25
const LEN_CONN = 14
const LEN_NODES = 32

type IncomingData struct {
	Connections int
	NodesTotal  int
	NodesGood   int
	NodesDead   int32
	NodesQueued int
	Log         string
	Msg         string
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
	buffMsgs        *buff.GuiBuffer
}

func New(ctx context.Context, ch chan IncomingData) *GUI {
	g := GUI{
		ctx:             ctx,
		ch:              ch,
		buffConnections: buff.New(LEN_CONN),
		buffNodesTotal:  buff.New(LEN_NODES),
		buffNodesQueued: buff.New(LEN_NODES),
		buffNodesGood:   buff.New(LEN_NODES),
		buffNodesDead:   buff.New(LEN_NODES),
		buffLogs:        buff.New(LEN_LOGS),
		buffMsgs:        buff.New(LEN_LOGS),
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
			g.buffNodesDead.AddNum(int(d.NodesDead))
			g.buffLogs.AddString(d.Log)
			g.buffMsgs.AddString(d.Msg)
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
	progress := widgets.NewGauge()
	progress.Title = "Progress"
	progress.Percent = 0
	progress.BarColor = tui.ColorBlue
	progress.BorderStyle.Fg = tui.ColorWhite
	progress.Label = "Loading..."
	progress.LabelStyle = tui.NewStyle(tui.ColorWhite)

	// CONNECTIONS
	chartConn := widgets.NewSparkline()
	// max connections
	chartConn.MaxVal = float64(cfg.ConnectionsLimit)
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
	chartNodesTotal.Data = [][]float64{make([]float64, LEN_NODES)}
	chartNodesTotal.LineColors = []tui.Color{tui.ColorWhite} // force the collor, bug

	// QUEUE
	chartNodesQueue := widgets.NewPlot()
	chartNodesQueue.ShowAxes = false
	chartNodesQueue.Data = [][]float64{make([]float64, LEN_NODES)}
	chartNodesQueue.LineColors = []tui.Color{tui.ColorYellow} // force the collor, bug

	// good
	chartNodesGood := widgets.NewPlot()
	chartNodesGood.ShowAxes = false
	chartNodesGood.Data = [][]float64{make([]float64, LEN_NODES)}
	chartNodesGood.LineColors = []tui.Color{tui.ColorGreen} // force the collor, bug

	// dead
	chartNodesDead := widgets.NewPlot()
	chartNodesDead.ShowAxes = false
	chartNodesDead.Data = [][]float64{make([]float64, LEN_NODES)}
	chartNodesDead.LineColors = []tui.Color{tui.ColorRed} // force the collor, bug

	// LOGS
	log := widgets.NewParagraph()
	log.WrapText = true
	log.Text = "Loading..."
	log.Title = "Logs"

	// MSG
	msg := widgets.NewParagraph()
	msg.WrapText = true
	msg.Text = "Connecting..."
	msg.Title = "Messages"

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
			tui.NewCol(0.45, log),
			tui.NewCol(0.45, msg),
			tui.NewCol(0.1, chartConnWrap),
		),
		// progress
		tui.NewRow(0.1,
			tui.NewCol(1, progress),
		),
	)
	tui.Render(grid)

	// send debug data
	if os.Getenv("GUI_DEBUG") == "1" {
		go g.sendDebugData()
	}

	// UPDATER
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

			// update logs
			log.Text = g.buffLogs.GetString()
			msg.Text = g.buffMsgs.GetString()

			// connections update
			chartConnWrap.Sparklines[0].Data = g.buffConnections.GetFloats()

			// calc progress
			total := g.buffNodesTotal.GetLastNum()
			queued := g.buffNodesQueued.GetLastNum()
			good := g.buffNodesGood.GetLastNum()
			dead := g.buffNodesDead.GetLastNum()
			left := good + dead
			if left > 0 {
				prog := float64(left) / float64(total) * 100
				progress.Percent = int(prog)
				progress.Label = fmt.Sprintf("%.0f%%", prog)
			} else if queued == 0 {
				progress.Label = "Loading seeds..."
			} else {
				progress.Label = "Connecting..."
			}

			// update charts
			chartNodesTotal.Data[0] = g.buffNodesTotal.GetFloats()
			chartNodesQueue.Data[0] = g.buffNodesQueued.GetFloats()
			chartNodesGood.Data[0] = g.buffNodesGood.GetFloats()
			chartNodesDead.Data[0] = g.buffNodesDead.GetFloats()

			//  update titles
			updateTitle(chartNodesTotal, total, "Total")
			updateTitle(chartNodesQueue, queued, "Queue")
			updateTitle(chartNodesGood, good, "Good")
			updateTitle(chartNodesDead, dead, "Dead")
			updateTitleChart(chartConnWrap, g.buffConnections.GetLastNum(), "Conn.")

			// update info
			stats.Rows = g.getInfo()

			// debug info to logs
			if os.Getenv("GUI_MEM") == "1" {
				text := fmt.Sprintf("buffNodesTotal: len %d, cap %d\n", len(g.buffNodesTotal.GetFloats()), cap(g.buffNodesTotal.GetFloats()))
				text += fmt.Sprintf("buffNodesQueued: len %d, cap %d\n", len(g.buffNodesQueued.GetFloats()), cap(g.buffNodesQueued.GetFloats()))
				text += fmt.Sprintf("buffNodesGood: len %d, cap %d\n", len(g.buffNodesGood.GetFloats()), cap(g.buffNodesGood.GetFloats()))
				text += fmt.Sprintf("buffNodesDead: len %d, cap %d\n", len(g.buffNodesDead.GetFloats()), cap(g.buffNodesDead.GetFloats()))
				text += fmt.Sprintf("buffConnections: len %d, cap %d\n", len(g.buffConnections.GetFloats()), cap(g.buffConnections.GetFloats()))

				// msg += fmt.Sprintf("dataNodesTotalLL: %d\n", g.dataNodesTotalList.Len())
				// report G count and memory used
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				text += fmt.Sprintf("STATS: G:%d, MEM:%dKb\n", runtime.NumGoroutine(), m.Alloc/1024)
				msg.Text = text
			}
			tui.Render(grid)
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
			cnt++
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
				NodesDead:   int32(rDead),
				Log:         fmt.Sprintf("test log %d", cnt),
				Msg:         fmt.Sprintf("test msg %d", cnt),
			}
		}
	}
}
