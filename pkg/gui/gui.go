package gui

import (
	"container/list"
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"strings"
	"sync"
	"time"

	tui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cfg = config.New()
var mu sync.Mutex = sync.Mutex{}

const lenLogs = 20
const lenConnChart = 30
const lenNodesChart = 40

type Data struct {
	Connections int
	NodesTotal  int
	NodesGood   int
	NodesDead   int
	NodesQueued int
	MsgIn       int
	MsgOut      int
}

type GUI struct {
	maxConnections int
	// Info table
	infoNodesTotal  int
	infoNodesGood   int
	infoNodesDead   int
	infoNodesQueued int
	infoConnections int
	infoMsgIn       int
	infoMsgOut      int

	// Connections chart
	dataConnectionsList *list.List
	dataConnections     []float64

	// Nodes chart
	// linked list to update
	dataNodesTotalList *list.List
	dataNodesGoodList  *list.List
	dataNodesDeadList  *list.List
	// slices for the chart, convert from linked list
	dataNodesTotal []float64
	dataNodesGood  []float64
	dataNodesDead  []float64

	logsList *list.List
	logs     []string
}

func New() *GUI {
	g := GUI{
		maxConnections: cfg.ConnectionsLimit,

		dataConnectionsList: list.New(),
		dataConnections:     make([]float64, lenConnChart),

		dataNodesTotalList: list.New(),
		dataNodesGoodList:  list.New(),
		dataNodesDeadList:  list.New(),
		dataNodesTotal:     make([]float64, lenNodesChart),
		dataNodesGood:      make([]float64, lenNodesChart),
		dataNodesDead:      make([]float64, lenNodesChart),

		logsList: list.New(),
		logs:     make([]string, lenLogs),
	}
	return &g
}

func (g *GUI) Start() {
	if err := tui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer tui.Close()

	// PROGRESS
	g0 := widgets.NewGauge()
	g0.Title = "Progress"
	g0.Percent = 20
	g0.BarColor = tui.ColorGreen
	g0.BorderStyle.Fg = tui.ColorWhite
	g0.Label = fmt.Sprintf("%d/%d", 20, 100)
	g0.LabelStyle = tui.NewStyle(tui.ColorWhite)

	// CONNECTIONS
	chartConn := widgets.NewSparkline()
	// max connections
	chartConn.MaxVal = float64(g.maxConnections)
	chartConn.Data = []float64{0}
	chartConn.LineColor = tui.ColorMagenta
	chartConn.TitleStyle.Fg = tui.ColorWhite
	chartConnGroup := widgets.NewSparklineGroup(chartConn)
	chartConnGroup.Title = "Connections"

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
			tui.NewCol(0.2, chartConnGroup), // conn
			tui.NewCol(0.2, stats),          // stats
			tui.NewCol(0.2, chartNodesQueue),
			tui.NewCol(0.2, chartNodesGood),
			tui.NewCol(0.2, chartNodesDead),
		),
		// progress
		tui.NewRow(0.1,
			tui.NewCol(1, g0),
		),
		// logs
		tui.NewRow(0.65,
			tui.NewCol(1.0, p),
		),
	)
	tui.Render(grid)

	// UPDATER
	tickerCount := 1
	uiEvents := tui.PollEvents()
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
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
			p.Text = strings.Join(g.logs, "\n")

			// connections update
			chartConnGroup.Sparklines[0].Data = g.dataConnections

			// nodes chart
			chartNodesQueue.Data[0] = g.dataNodesTotal
			chartNodesGood.Data[0] = g.dataNodesGood
			chartNodesDead.Data[0] = g.dataNodesDead

			//  update titles
			g.updateTitle(chartNodesQueue, "Queue")
			g.updateTitle(chartNodesGood, "Good")
			g.updateTitle(chartNodesDead, "Dead")

			// update info
			stats.Rows = g.getInfo()

			// debug info
			// msg := fmt.Sprintf("dataNodesTotal: len %d, cap %d\n", len(g.dataNodesTotal), cap(g.dataNodesTotal))
			// msg += fmt.Sprintf("dataNodesTotalLL: %d\n", g.dataNodesTotalList.Len())
			// msg += fmt.Sprintf("chartQueue.Data len: %d, cap: %d\n", len(chartNodesQueue.Data), cap(chartNodesQueue.Data))
			// msg += fmt.Sprintf("chartQueue.Data[0] len: %d, cap: %d\n", len(chartNodesQueue.Data[0]), cap(chartNodesQueue.Data[0]))
			// msg += fmt.Sprintf("chartConn: len %d, cap %d\n", len(chartConnGroup.Sparklines[0].Data), cap(chartConnGroup.Sparklines[0].Data))
			// msg += fmt.Sprintf("chartConn data len: %d, cap: %d\n", len(g.dataConnections), cap(g.dataConnections))
			// msg += fmt.Sprint("data: ", g.dataConnections)
			// p.Text = msg
			tui.Render(grid)
			tickerCount++
		}
	}
}

// update GUI data
// in serial data (charts) we push new data to the linked lists first
// and then construct slices from the linked lists
func (g *GUI) Update(d Data) {
	mu.Lock()
	// update connection data
	updateDataList(g.dataConnectionsList, float64(d.Connections), lenConnChart)
	updateSlice(g.dataConnections, g.dataConnectionsList, lenConnChart)
	g.infoConnections = d.Connections

	// update nodes linked lists (in place)
	updateDataList(g.dataNodesTotalList, float64(d.NodesTotal), lenNodesChart)
	updateDataList(g.dataNodesGoodList, float64(d.NodesGood), lenNodesChart)
	updateDataList(g.dataNodesDeadList, float64(d.NodesDead), lenNodesChart)

	// update slices in placj
	updateSlice(g.dataNodesTotal, g.dataNodesTotalList, lenNodesChart)
	updateSlice(g.dataNodesGood, g.dataNodesGoodList, lenNodesChart)
	updateSlice(g.dataNodesDead, g.dataNodesDeadList, lenNodesChart)

	if d.Connections > 0 {
		g.infoConnections = d.Connections
	}
	if d.NodesTotal > 0 {
		g.infoNodesTotal = d.NodesTotal
	}
	if d.NodesQueued > 0 {
		g.infoNodesQueued = d.NodesQueued
	}
	if d.NodesGood > 0 {
		g.infoNodesGood = d.NodesGood
	}
	if d.NodesDead > 0 {
		g.infoNodesDead = d.NodesDead
	}
	if d.MsgIn > 0 {
		g.infoMsgIn = d.MsgIn
	}
	if d.MsgOut > 0 {
		g.infoMsgOut = d.MsgOut
	}

	mu.Unlock()
}

func updateDataList[T any](l *list.List, data T, limit int) {
	switch v := any(data).(type) {
	case float64:
		if v == 0 {
			return
		}
	case string:
		if v == "" {
			return
		}
	}
	l.PushBack(T(data))
	if l.Len() > limit {
		l.Remove(l.Front())
	}
}

func updateSlice[T any](s []T, l *list.List, limit int) {
	i := 0
	// loop from back to front and update slice accordingly
	for e := l.Back(); e != nil; e = e.Prev() {
		if i >= limit {
			break
		}
		idx := limit - 1 - i
		if idx >= len(s) {
			break
		}
		s[idx] = e.Value.(T)
		i++
	}
}

func (g *GUI) Log(log string) {
	mu.Lock()
	updateDataList(g.logsList, log, lenLogs)
	updateSlice(g.logs, g.logsList, lenLogs)
	// g.logs = append(g.logs, log)
	// if len(g.logs) > lenLogs {
	// 	g.logs = g.logs[len(g.logs)-lenLogs:]
	// }
	mu.Unlock()
}

// func updateLogSlice(s []string, l *list.List, limit int) {
// 	i := 0
// 	// loop from back to front and update slice accordingly
// 	for e := l.Back(); e != nil; e = e.Prev() {
// 		if i >= limit {
// 			break
// 		}
// 		idx := limit - 1 - i
// 		if idx >= len(s) {
// 			break
// 		}
// 		s[idx] = e.Value.(string)
// 		i++
// 	}
// }
//
// func updateLogsList(l *list.List, log string, limit int) {
// 	if log == "" {
// 		return
// 	}
// 	l.PushBack(log)
// 	if l.Len() > limit {
// 		l.Remove(l.Front())
// 	}
// }

func (g *GUI) updateTitle(chart *widgets.Plot, title string) {
	if len(g.dataNodesDead) > 0 {
		title += fmt.Sprintf(" (%.0f)", g.dataNodesDead[len(g.dataNodesDead)-1])
	}
	chart.Title = title
}

func (g *GUI) getInfo() [][]string {
	return [][]string{
		{"Total nodes", fmt.Sprintf("%d", g.infoNodesTotal)},
		{"Good nodes", fmt.Sprintf("%d", g.infoNodesGood)},
		{"Dead nodes", fmt.Sprintf("%d", g.infoNodesDead)},
		{"Queue", fmt.Sprintf("%d", g.infoNodesQueued)},
		{"Connections", fmt.Sprintf("%d/%d", g.infoConnections, g.maxConnections)},
		{"Msg out", fmt.Sprintf("%d", g.infoMsgOut)},
		{"Msg in", fmt.Sprintf("%d", g.infoMsgIn)},
	}
}
