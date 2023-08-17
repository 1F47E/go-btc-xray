package gui

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/1F47E/go-btc-xray/internal/config"

	tui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cfg = config.New()

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
	buffConnections []float64
	buffNodesTotal  []float64
	buffNodesQueued []float64
	buffNodesGood   []float64
	buffNodesDead   []float64
	buffLogs        []string
	buffMsgs        []string
}

func New(ctx context.Context, ch chan IncomingData) *GUI {
	g := GUI{
		ctx:             ctx,
		ch:              ch,
		buffConnections: make([]float64, LEN_CONN),
		buffNodesTotal:  make([]float64, LEN_NODES),
		buffNodesQueued: make([]float64, LEN_NODES),
		buffNodesGood:   make([]float64, LEN_NODES),
		buffNodesDead:   make([]float64, LEN_NODES),
		buffLogs:        make([]string, LEN_LOGS),
		buffMsgs:        make([]string, LEN_LOGS),
	}
	return &g
}

func (g *GUI) listner() {
	for {
		select {
		case <-g.ctx.Done():
			return
		case d := <-g.ch:
			g.buffConnections = buffAddFloat(g.buffConnections, float64(d.Connections))
			g.buffNodesTotal = buffAddFloat(g.buffNodesTotal, float64(d.NodesTotal))
			g.buffNodesQueued = buffAddFloat(g.buffNodesQueued, float64(d.NodesQueued))
			g.buffNodesGood = buffAddFloat(g.buffNodesGood, float64(d.NodesGood))
			g.buffNodesDead = buffAddFloat(g.buffNodesDead, float64(d.NodesDead))
			g.buffLogs = buffAddString(g.buffLogs, d.Log)
			g.buffMsgs = buffAddString(g.buffMsgs, d.Msg)
		}
	}
}

func buffAddFloat(buff []float64, v float64) []float64 {
	if v == 0 {
		return buff
	}
	buff = append(buff, v)
	buff = buff[1:]
	return buff
}

func buffAddString(buff []string, v string) []string {
	if v == "" {
		return buff
	}
	buff = append(buff, v)
	buff = buff[1:]
	return buff
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
			log.Text = strings.Join(g.buffLogs, "\n")
			msg.Text = strings.Join(g.buffMsgs, "\n")

			// connections update
			chartConnWrap.Sparklines[0].Data = g.buffConnections

			// calc progress
			conn := g.buffConnections[LEN_CONN-1]
			total := g.buffNodesTotal[LEN_NODES-1]
			queued := g.buffNodesQueued[LEN_NODES-1]
			good := g.buffNodesGood[LEN_NODES-1]
			dead := g.buffNodesDead[LEN_NODES-1]
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
			chartNodesTotal.Data[0] = g.buffNodesTotal
			chartNodesQueue.Data[0] = g.buffNodesQueued
			chartNodesGood.Data[0] = g.buffNodesGood
			chartNodesDead.Data[0] = g.buffNodesDead

			//  update titles
			updateTitlePlot(chartNodesTotal, total, "Total")
			updateTitlePlot(chartNodesQueue, queued, "Queue")
			updateTitlePlot(chartNodesGood, good, "Good")
			updateTitlePlot(chartNodesDead, dead, "Dead")
			updateTitleChart(chartConnWrap, conn, "Conn.")

			// update info
			stats.Rows = g.getInfo()

			// debug info to logs
			if os.Getenv("GUI_MEM") == "1" {
				text := fmt.Sprintf("buffNodesTotal: len %d, cap %d\n", len(g.buffNodesTotal), cap(g.buffNodesTotal))
				text += fmt.Sprintf("buffNodesQueued: len %d, cap %d\n", len(g.buffNodesQueued), cap(g.buffNodesQueued))
				text += fmt.Sprintf("buffNodesGood: len %d, cap %d\n", len(g.buffNodesGood), cap(g.buffNodesGood))
				text += fmt.Sprintf("buffNodesDead: len %d, cap %d\n", len(g.buffNodesDead), cap(g.buffNodesDead))
				text += fmt.Sprintf("buffConnections: len %d, cap %d\n", len(g.buffConnections), cap(g.buffConnections))

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

func (g *GUI) getInfo() [][]string {
	return [][]string{
		{"Total nodes", fmt.Sprintf("%.0f", g.buffNodesTotal[LEN_NODES-1])},
		{"Good nodes", fmt.Sprintf("%.0f", g.buffNodesGood[LEN_NODES-1])},
		{"Dead nodes", fmt.Sprintf("%.0f", g.buffNodesDead[LEN_NODES-1])},
		{"Queue", fmt.Sprintf("%.0f", g.buffNodesQueued[LEN_NODES-1])},
		{"Connections", fmt.Sprintf("%.0f/%d", g.buffConnections[LEN_CONN-1], cfg.ConnectionsLimit)},
	}
}

// update titles
func updateTitleChart(chart *widgets.SparklineGroup, data float64, title string) {
	if data > 0 {
		title += fmt.Sprintf(": %.0f", data)
	}
	chart.Title = title
}

func updateTitlePlot(chart *widgets.Plot, data float64, title string) {
	if data > 0 {
		title += fmt.Sprintf(" (%.0f)", data)
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
