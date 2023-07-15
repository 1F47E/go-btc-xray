package gui

import (
	"container/list"
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"os"
	"runtime"
	"strings"
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
	dataNodesTotalList  *list.List
	dataNodesQueuedList *list.List
	dataNodesGoodList   *list.List
	dataNodesDeadList   *list.List
	// slices for the chart, convert from linked list
	dataNodesTotal  []float64
	dataNodesQueued []float64
	dataNodesGood   []float64
	dataNodesDead   []float64

	logsList *list.List
	logs     []string
}

func New() *GUI {
	g := GUI{
		maxConnections: cfg.ConnectionsLimit,

		dataConnectionsList: list.New(),
		dataConnections:     make([]float64, lenConnChart),

		dataNodesTotalList:  list.New(),
		dataNodesQueuedList: list.New(),
		dataNodesGoodList:   list.New(),
		dataNodesDeadList:   list.New(),
		dataNodesTotal:      make([]float64, lenNodesChart),
		dataNodesQueued:     make([]float64, lenNodesChart),
		dataNodesGood:       make([]float64, lenNodesChart),
		dataNodesDead:       make([]float64, lenNodesChart),

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
	g0.BarColor = tui.ColorBlue
	g0.BorderStyle.Fg = tui.ColorWhite
	g0.Label = fmt.Sprintf("%d/%d", 20, 100)
	g0.LabelStyle = tui.NewStyle(tui.ColorWhite)

	// CONNECTIONS
	chartConn := widgets.NewSparkline()
	// max connections
	chartConn.MaxVal = float64(g.maxConnections) * 1.2 // height hack
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
			chartConnWrap.Sparklines[0].Data = g.dataConnections

			// nodes chart
			// chartNodesTotal.Data[0] = g.dataNodesTotal.Data()
			chartNodesTotal.Data[0] = g.dataNodesTotal
			chartNodesQueue.Data[0] = g.dataNodesQueued
			chartNodesGood.Data[0] = g.dataNodesGood
			chartNodesDead.Data[0] = g.dataNodesDead

			//  update titles
			updateTitle(g.infoNodesTotal, chartNodesTotal, "Total")
			updateTitle(g.infoNodesQueued, chartNodesQueue, "Queue")
			updateTitle(g.infoNodesGood, chartNodesGood, "Good")
			updateTitle(g.infoNodesDead, chartNodesDead, "Dead")
			updateTitleChart(g.infoConnections, chartConnWrap, "Conn.")

			// update info
			stats.Rows = g.getInfo()

			// debug info to logs
			if os.Getenv("LOGS") == "2" {
				msg := fmt.Sprintf("dataNodesTotal: len %d, cap %d\n", len(g.dataNodesTotal), cap(g.dataNodesTotal))
				msg += fmt.Sprintf("dataNodesTotalLL: %d\n", g.dataNodesTotalList.Len())
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
func (g *GUI) Update(d Data) {

	// update nodes linked lists (in place)
	mu.Lock()
	if d.Connections > 0 {
		g.infoConnections = d.Connections
		updateDataList(g.dataConnectionsList, float64(d.Connections), g.dataConnections, lenConnChart)
	}
	if d.NodesTotal > 0 {
		g.infoNodesTotal = d.NodesTotal
		updateDataList(g.dataNodesTotalList, float64(d.NodesTotal), g.dataNodesTotal, lenNodesChart)
	}
	if d.NodesQueued > 0 {
		g.infoNodesQueued = d.NodesQueued
		updateDataList(g.dataNodesQueuedList, float64(d.NodesQueued), g.dataNodesQueued, lenNodesChart)
	}
	if d.NodesGood > 0 {
		g.infoNodesGood = d.NodesGood
		updateDataList(g.dataNodesGoodList, float64(d.NodesGood), g.dataNodesGood, lenNodesChart)
	}
	if d.NodesDead > 0 {
		g.infoNodesDead = d.NodesDead
		updateDataList(g.dataNodesDeadList, float64(d.NodesDead), g.dataNodesDead, lenNodesChart)
	}
	if d.MsgIn > 0 {
		g.infoMsgIn = d.MsgIn
	}
	if d.MsgOut > 0 {
		g.infoMsgOut = d.MsgOut
	}
	mu.Unlock()

}

func (g *GUI) Log(log string) {
	updateDataList(g.logsList, log, g.logs, lenLogs)
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

func updateDataList[T any](l *list.List, val T, mirror []T, limit int) {
	switch v := any(val).(type) {
	case float64:
		if v == 0 {
			return
		}
	case string:
		if v == "" {
			return
		}
	}
	l.PushBack(T(val))
	if l.Len() > limit {
		l.Remove(l.Front())
	}
	// copy list elements to the slice
	updateSlice(mirror, l, limit)
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

func updateTitleChart(data int, chart *widgets.SparklineGroup, title string) {
	if data > 0 {
		title += fmt.Sprintf(": %d", data)
	}
	chart.Title = title
}

func updateTitle(data int, chart *widgets.Plot, title string) {
	if data > 0 {
		title += fmt.Sprintf(" (%d)", data)
	}
	chart.Title = title
}
