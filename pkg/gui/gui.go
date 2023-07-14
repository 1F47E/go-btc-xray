package gui

import (
	"container/list"
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"math"
	"sync"
	"time"

	tui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cfg = config.New()
var mu sync.Mutex = sync.Mutex{}

const limitLogs = 20
const limitConn = 30
const lenNodesChart = 40

type Data struct {
	Connections int
	NodesTotal  int
	NodesGood   int
	NodesDead   int
}

type GUI struct {
	dataConnections []float64
	// linked list to update
	dataNodesTotalList *list.List
	dataNodesGoodList  *list.List
	dataNodesDeadList  *list.List
	// slices for the chart, convert from linked list
	dataNodesTotal []float64
	dataNodesGood  []float64
	dataNodesDead  []float64
	maxConnections int
	logs           []string
}

func New() *GUI {
	g := GUI{
		maxConnections:     cfg.ConnectionsLimit,
		dataConnections:    make([]float64, limitConn),
		dataNodesTotalList: list.New(),
		dataNodesGoodList:  list.New(),
		dataNodesDeadList:  list.New(),
		dataNodesTotal:     make([]float64, lenNodesChart),
		dataNodesGood:      make([]float64, lenNodesChart),
		dataNodesDead:      make([]float64, lenNodesChart),
	}

	// fake data debug
	// k := 0
	// for i := 0; i < lenNodesChart; i++ {
	// 	step := 5
	// 	if i%step == 0 {
	// 		k = k + step
	// 	}
	// 	// update from the end
	// 	idx := len(g.dataNodesTotal) - 1 - i
	// 	g.dataNodesTotal[idx] = float64(i)
	// }

	return &g
}

func (g *GUI) convertToSlice(l *list.List) []float64 {
	// convert from linked list to list of float64
	data := make([]float64, lenNodesChart)
	i := 0
	for e := l.Back(); e != nil; e = e.Prev() {
		if i >= lenNodesChart {
			break
		}
		idx := lenNodesChart - 1 - i
		data[idx] = e.Value.(float64)
		i++
	}
	return data
}

// TODO: optimize
func (g *GUI) Update(d Data) {
	mu.Lock()
	if d.Connections > 0 {
		g.dataConnections = append(g.dataConnections, float64(d.Connections))
		if len(g.dataConnections) > limitConn {
			g.dataConnections = g.dataConnections[len(g.dataConnections)-limitConn:]
		}
	}
	if d.NodesTotal > 0 {
		// push to the back
		g.dataNodesTotalList.PushBack(float64(d.NodesTotal))
		// remove from the front
		if g.dataNodesTotalList.Len() > lenNodesChart {
			g.dataNodesTotalList.Remove(g.dataNodesTotalList.Front())
		}
		g.dataNodesTotal = g.convertToSlice(g.dataNodesTotalList)
	}
	if d.NodesGood > 0 {
		// push to the back
		g.dataNodesGoodList.PushBack(float64(d.NodesGood))
		// remove from the front
		if g.dataNodesGoodList.Len() > lenNodesChart {
			g.dataNodesGoodList.Remove(g.dataNodesGoodList.Front())
		}
		g.dataNodesGood = g.convertToSlice(g.dataNodesGoodList)
	}
	if d.NodesDead > 0 {
		// push to the back
		g.dataNodesDeadList.PushBack(float64(d.NodesDead))
		// remove from the front
		if g.dataNodesDeadList.Len() > lenNodesChart {
			g.dataNodesDeadList.Remove(g.dataNodesDeadList.Front())
		}
		g.dataNodesDead = g.convertToSlice(g.dataNodesDeadList)
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
	if err := tui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer tui.Close()

	// fake data
	sinFloat64A := (func() []float64 {
		n := 400
		data := make([]float64, n)
		for i := range data {
			data[i] = 1 + math.Sin(float64(i)/5)
		}
		return data
	})()

	// PROGRESS
	g0 := widgets.NewGauge()
	g0.Title = "Progress"
	g0.Percent = 20
	g0.BarColor = tui.ColorGreen
	g0.BorderStyle.Fg = tui.ColorWhite
	g0.Label = fmt.Sprintf("%d/%d", 20, 100)
	g0.LabelStyle = tui.NewStyle(tui.ColorWhite)

	// CONNECTIONS
	sl := widgets.NewSparkline()
	// max connections
	sl.MaxVal = float64(g.maxConnections)
	sl.Data = sinFloat64A[:100]
	sl.LineColor = tui.ColorMagenta
	sl.TitleStyle.Fg = tui.ColorWhite
	slg := widgets.NewSparklineGroup(sl)
	slg.Title = "Connections"

	// STATS
	t1 := widgets.NewTable()
	t1.RowSeparator = false
	t1.FillRow = false
	t1.RowStyles[1] = tui.NewStyle(tui.ColorGreen)
	t1.RowStyles[2] = tui.NewStyle(tui.ColorRed)
	t1.RowStyles[3] = tui.NewStyle(tui.ColorYellow)
	t1.RowStyles[4] = tui.NewStyle(tui.ColorMagenta)
	t1.Rows = [][]string{
		{"Total nodes", "10000"},
		{"Good nodes", "123"},
		{"Dead nodes", "456"},
		{"Wait list", "789"},
		{"Connections", fmt.Sprintf("%d/%d", 0, g.maxConnections)},
		{"Msg out", "123"},
		{"Msg in", "50"},
	}
	t1.TextStyle = tui.NewStyle(tui.ColorWhite)
	tui.Render(t1)

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
		tui.NewRow(1.0/2-1.0/5,
			tui.NewCol(0.2, slg), // conn
			tui.NewCol(0.2, t1),  // stats
			tui.NewCol(0.2, chartNodesQueue),
			tui.NewCol(0.2, chartNodesGood),
			tui.NewCol(0.2, chartNodesDead),
		),
		// progress
		tui.NewRow(1.0/10,
			tui.NewCol(1, g0),
		),
		// logs
		tui.NewRow(1.0/2,
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

			// p.Text = strings.Join(g.logs, "\n")

			// connections update
			slg.Sparklines[0].Data = g.dataConnections

			// nodes chart
			chartNodesQueue.Data[0] = g.dataNodesTotal
			chartNodesGood.Data[0] = g.dataNodesGood
			chartNodesDead.Data[0] = g.dataNodesDead

			//  update titles
			g.updateTitle(chartNodesQueue, "Queue")
			g.updateTitle(chartNodesGood, "Good")
			g.updateTitle(chartNodesDead, "Dead")

			// debug info
			msg := fmt.Sprintf("dataNodesTotal: len %d, cap %d\n", len(g.dataNodesTotal), cap(g.dataNodesTotal))
			msg += fmt.Sprintf("dataNodesTotalLL: %d\n", g.dataNodesTotalList.Len())
			msg += fmt.Sprintf("chartQueue.Data len: %d, cap: %d\n", len(chartNodesQueue.Data), cap(chartNodesQueue.Data))
			msg += fmt.Sprintf("chartQueue.Data[0] len: %d, cap: %d\n", len(chartNodesQueue.Data[0]), cap(chartNodesQueue.Data[0]))
			msg += fmt.Sprint("data: ", g.dataNodesTotal)
			p.Text = msg
			tui.Render(grid)
			tickerCount++
		}
	}
}

func (g *GUI) updateTitle(chart *widgets.Plot, title string) {
	if len(g.dataNodesDead) > 0 {
		title += fmt.Sprintf(" (%.0f)", g.dataNodesDead[len(g.dataNodesDead)-1])
	}
	chart.Title = title
}
