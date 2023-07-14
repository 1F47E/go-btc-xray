package gui

import (
	"container/list"
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"math"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cfg = config.New()
var mu sync.Mutex = sync.Mutex{}

const limitLogs = 20
const limitConn = 30
const limitNodes = 100
const lenNodesChart = 75

type Data struct {
	Connections int
	NodesTotal  int
	NodesGood   int
	NodesDead   int
}

type GUI struct {
	dataConnections []float64
	dataNodesTotal  []float64
	// linked list
	dataNodesTotalLL *list.List
	dataNodesGood    []float64
	dataNodesDead    []float64
	maxConnections   int
	logs             []string
}

func New() *GUI {
	g := GUI{
		maxConnections:   cfg.ConnectionsLimit,
		dataConnections:  make([]float64, limitConn),
		dataNodesTotal:   make([]float64, lenNodesChart),
		dataNodesTotalLL: list.New(),
		dataNodesGood:    make([]float64, lenNodesChart),
		dataNodesDead:    make([]float64, lenNodesChart),
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
	// convert from linked list to float64
	data := make([]float64, lenNodesChart)
	i := 0
	for e := l.Front(); e != nil; e = e.Next() {
		data[i] = e.Value.(float64)
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
		// cut head first
		// if len(g.dataNodesTotal) > limitNodes {
		// 	g.dataNodesTotal = g.dataNodesTotal[len(g.dataNodesTotal)-limitNodes:]
		// }
		// g.dataNodesTotal = append(g.dataNodesTotal, float64(d.NodesTotal))
		// push
		// fmt.Printf("update total nodes with %d\n", d.NodesTotal)
		g.dataNodesTotalLL.PushBack(float64(d.NodesTotal))
		// remove from the front
		if g.dataNodesTotalLL.Len() > lenNodesChart {
			g.dataNodesTotalLL.Remove(g.dataNodesTotalLL.Front())
		}
		i := 0
		// for e := g.dataNodesTotalLL.Front(); e != nil; e = e.Next() {
		for e := g.dataNodesTotalLL.Back(); e != nil; e = e.Prev() {
			if i >= lenNodesChart {
				break
			}
			idx := lenNodesChart - 1 - i
			g.dataNodesTotal[idx] = e.Value.(float64)
			i++
		}
		// g.dataNodesTotal = g.convertToSlice(g.dataNodesTotalLL)
		// fmt.Printf("list len: %d\n", g.dataNodesTotalLL.Len())
	}
	// if d.NodesGood > 0 {
	// 	// cut head first
	// 	if len(g.dataNodesGood) > limitNodes {
	// 		g.dataNodesGood = g.dataNodesGood[len(g.dataNodesGood)-limitNodes:]
	// 	}
	// 	g.dataNodesGood = append(g.dataNodesGood, float64(d.NodesGood))
	// }
	//
	// if d.NodesDead > 0 {
	// 	// cut head first
	// 	if len(g.dataNodesDead) > limitNodes {
	// 		g.dataNodesDead = g.dataNodesDead[len(g.dataNodesDead)-limitNodes:]
	// 	}
	// 	g.dataNodesDead = append(g.dataNodesDead, float64(d.NodesDead))
	// }
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
	chartQueue.ShowAxes = true
	chartQueue.Title = "Nodes"
	// data is [][]float64
	// bug

	// chartQueue.Data = make([][]float64, 3)
	// for i := range chartQueue.Data {
	// 	chartQueue.Data[i] = make([]float64, 0)
	// 	copy(chartQueue.Data[i], g.dataConnections)
	// }

	// chartQueue.Data[0] = g.dataConnections
	// chartQueue.Data[1] = g.dataConnections
	// chartQueue.Data[2] = g.dataConnections
	// working
	// chartQueue.Data = append(chartQueue.Data, make([]float64, lenNodesChart))
	// chartQueue.Data = append(chartQueue.Data, g.dataNodesGood)
	// chartQueue.Data = append(chartQueue.Data, g.dataNodesDead)
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
	ticker := time.NewTicker(200 * time.Millisecond)
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
		case <-ticker.C:
			// if tickerCount == 100 {
			// 	return
			// }
			for _, g := range gs {
				g.Percent = (g.Percent + 3) % 100
			}
			// update logs

			// p.Text = strings.Join(g.logs, "\n")

			// connections update
			// slg.Sparklines[0].Data = sinFloat64B[tickerCount : tickerCount+100]
			slg.Sparklines[0].Data = g.dataConnections
			// slg.Sparklines[1].Data = sinFloat64B[tickerCount : tickerCount+100]
			// chartConn.Data[0] = sinFloat64A[2*tickerCount:]
			// chartQueue.Data[0] = sinFloat64B[2*tickerCount:]

			// chartQueue.Data[0] = g.dataNodesTotal
			// chartQueue.Data[1] = g.dataNodesGood
			// chartQueue.Data[2] = g.dataNodesDead

			data := sinFloat64A[2*tickerCount:]
			chartQueue.Data[0] = data
			chartQueue.Data[1] = sinFloat64B[1*tickerCount:]
			chartQueue.Data[2] = g.dataNodesTotal
			msg := fmt.Sprintf("data chart len: %d, cap: %d\n", len(data), cap(data))
			msg += fmt.Sprintf("dataNodesTotal: len %d, cap %d\n", len(g.dataNodesTotal), cap(g.dataNodesTotal))
			msg += fmt.Sprintf("dataNodesTotalLL: %d\n", g.dataNodesTotalLL.Len())
			msg += fmt.Sprintf("chartQueue.Data len: %d, cap: %d\n", len(chartQueue.Data), cap(chartQueue.Data))
			msg += fmt.Sprintf("chartQueue.Data[0] len: %d, cap: %d\n", len(chartQueue.Data[0]), cap(chartQueue.Data[0]))
			msg += fmt.Sprintf("chartQueue.Data[1] len: %d, cap: %d\n", len(chartQueue.Data[1]), cap(chartQueue.Data[1]))
			msg += fmt.Sprintf("chartQueue.Data[2] len: %d, cap: %d\n", len(chartQueue.Data[2]), cap(chartQueue.Data[2]))
			msg += fmt.Sprint("data: ", g.dataNodesTotal)
			p.Text = msg
			// chartQueue.Data[2] = sinFloat64A[1*tickerCount:]
			// chartConn.LineColors[0] = ui.ColorMagenta
			// chartQueue.LineColors[0] = ui.ColorYellow
			// lc.Data[1] = sinFloat64B[2*tickerCount:]
			ui.Render(grid)
			tickerCount++
		}
	}
}
