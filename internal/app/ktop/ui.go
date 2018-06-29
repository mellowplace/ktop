package ktop

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jroimartin/gocui"
)

const (
	OrderMemHigh = 0x1
	OrderMemLow  = 0x2
	OrderCPUHigh = 0x3
	OrderCPULow  = 0x4
)

var (
	metricsClose    = make(chan int, 1)
	metricsReceive  = make(chan *SimplifiedPodMetricsList, 1)
	summaryReceive  = make(chan *KubeSummary, 1)
	errorChan       = make(chan error, 1)
	timerChan       = make(chan int, 1)
	podMetricsList  = &SimplifiedPodMetricsList{}
	kubeSummary     = &KubeSummary{ServerInfo: "connecting..."}
	kubeContextName = ""
	ordering        = OrderCPUHigh
)

func StartUI(kubeConfigFile, kubeContextName, namespace string) error {

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return err
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		return err
	}

	go collectStats(kubeConfigFile, kubeContextName, namespace, g, timerChan)
	go statsTimer()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}

	return nil
}

func collectStats(kubeConfigFile, kubeContextName, namespace string, g *gocui.Gui, timerChan chan int) {
	go pollKubeSummary(kubeConfigFile, kubeContextName, summaryReceive, metricsClose, errorChan)
	go pollPodMetrics(kubeConfigFile, kubeContextName, namespace, metricsClose, metricsReceive, errorChan)

	for {

		select {
		case list, open := <-metricsReceive:
			if open == false {
				return
			}
			podMetricsList = list
		case kubeSummary = <-summaryReceive:
			// don't need to do anything here, just set kube summary above
		case err := <-errorChan:
			g.Update(func(g *gocui.Gui) error {
				g.Close()
				return err
			})
			return
		}

		g.Update(func(g *gocui.Gui) error {

			maxX, _ := g.Size()

			v, err := g.View("help")
			if err != nil {
				return err
			}
			fmt.Fprintf(v, "%s %s %s %"+strconv.Itoa(maxX)+"s\n", "q = QUIT", "m/M = ORDER BY MEM HIGH/LOW", "c/C = ORDER BY CPU HIGH/LOW", " ")

			v, err = g.View("totals")

			if err != nil {
				return err
			}
			drawTotals(g, v)

			v, err = g.View("main")
			if err != nil {
				return err
			}
			v.Clear()

			format := "%-50s %-10s %-10s %-15s %-15s %-20s"
			// format header just makes sure we draw right across the available screen
			// to get the highlight all the way across
			formatHeader := format + "%" + strconv.FormatInt(int64(maxX), 10) + "s\n"
			v.Highlight = true
			v.SelFgColor = gocui.ColorBlack | gocui.AttrBold
			v.SelBgColor = gocui.ColorWhite

			fmt.Fprintf(v, formatHeader, "POD NAME", "CPU (used)", "CPU (limit)", "MEM (used)", "MEM (limit)", "Namespace", " ")

			switch ordering {
			case OrderMemHigh:
				podMetricsList.OrderByHighestMemUsage()
			case OrderMemLow:
				podMetricsList.OrderByLowestMemUsage()
			case OrderCPUHigh:
				podMetricsList.OrderByHighestCPUUsage()
			case OrderCPULow:
				podMetricsList.OrderByLowestCPUUsage()
			}

			for _, item := range podMetricsList.Pods {
				fmt.Fprintf(v, format+"\n", trimExcess(item.PodName, 50), item.CPUMillisString(), "-", item.MemoryBytesString(), "-", item.Namespace)
			}
			return nil
		})

		<-timerChan
	}
}

func drawTotals(g *gocui.Gui, v *gocui.View) error {

	v.Title = kubeSummary.ServerInfo

	v.Clear()
	format := "%40s %s\n"
	fmt.Fprintf(v, format, "Time:", podMetricsList.PollTime.Format(time.RFC850))
	fmt.Fprintf(v, format, "Total Nodes in Cluster:", strconv.Itoa(kubeSummary.TotalNodes))
	fmt.Fprintf(v, "%40s %10s / %s (%s)\n", "Memory Usage:", kubeSummary.GetTotalUsedMemory(), kubeSummary.GetTotalAllocatableMemory(), kubeSummary.GetMemPercentUsed())
	fmt.Fprintf(v, "%40s %10d / %d (%s)\n", "CPU Usage:", kubeSummary.TotalUsedCPUMillis, kubeSummary.TotalAllocatableCPUMillis, kubeSummary.GetCPUPercentUsed())
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("help", -1, maxY-2, maxX, maxY); err != nil {
		v.Frame = false
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Highlight = true
		v.SelFgColor = gocui.ColorWhite
		v.SelBgColor = gocui.ColorBlack
	}

	if v, err := g.SetView("totals", 1, 0, maxX-1, 6); err != nil {
		v.Frame = true
		if err != gocui.ErrUnknownView {
			return err
		}

		return drawTotals(g, v)

	}

	if v, err := g.SetView("main", 1, 6, maxX-1, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false

	}

	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeySpace, gocui.ModNone, pokeTimer); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'm', gocui.ModNone, orderByMemHigh); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'M', gocui.ModNone, orderByMemLow); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'c', gocui.ModNone, orderByCPUHigh); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'C', gocui.ModNone, orderByCPULow); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	metricsClose <- 1
	return gocui.ErrQuit
}

func pokeTimer(g *gocui.Gui, v *gocui.View) error {
	timerChan <- 1
	return nil
}

func orderByMemHigh(g *gocui.Gui, v *gocui.View) error {
	ordering = OrderMemHigh
	timerChan <- 1
	return nil
}

func orderByMemLow(g *gocui.Gui, v *gocui.View) error {
	ordering = OrderMemLow
	timerChan <- 1
	return nil
}

func orderByCPUHigh(g *gocui.Gui, v *gocui.View) error {
	ordering = OrderCPUHigh
	timerChan <- 1
	return nil
}

func orderByCPULow(g *gocui.Gui, v *gocui.View) error {
	ordering = OrderCPULow
	timerChan <- 1
	return nil
}

func statsTimer() {
	for {
		timerChan <- 1
		time.Sleep(2 * time.Second)
	}
}

func trimExcess(input string, maxLength int) string {
	if len(input) > maxLength {
		return input[0:maxLength]
	} else {
		return input
	}
}
