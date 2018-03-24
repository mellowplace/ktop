package ktop

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jroimartin/gocui"
)

var (
	metricsClose                             = make(chan int, 1)
	metricsReceive                           = make(chan *SimplifiedPodMetricsList, 1)
	errorChan                                = make(chan error, 1)
	timerChan                                = make(chan int, 1)
	podMetricsList *SimplifiedPodMetricsList = &SimplifiedPodMetricsList{}
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
		log.Panicln(err)
	}

	return nil
}

func collectStats(kubeConfigFile, kubeContextName, namespace string, g *gocui.Gui, timerChan chan int) {

	go pollPodMetrics(kubeConfigFile, kubeContextName, namespace, metricsClose, metricsReceive, errorChan)

	for {

		select {
		case list, open := <-metricsReceive:
			if open == false {
				return
			}
			podMetricsList = list
		case err := <-errorChan:
			g.Update(func(g *gocui.Gui) error {
				return err
			})
			return
		}

		g.Update(func(g *gocui.Gui) error {

			v, err := g.View("totals")

			if err != nil {
				return err
			}
			drawTotals(g, v)

			v, err = g.View("main")
			if err != nil {
				return err
			}
			v.Clear()

			maxX, _ := g.Size()
			format := "%50s %10s %10s %15s %15s"
			// format header just makes sure we draw right across the available screen
			// to get the highlight all the way across
			formatHeader := format + "%" + strconv.FormatInt(int64(maxX), 10) + "s\n"
			v.Highlight = true
			v.SelFgColor = gocui.ColorBlack | gocui.AttrBold
			v.SelBgColor = gocui.ColorWhite

			fmt.Fprintf(v, formatHeader, "POD NAME", "CPU (used)", "CPU (limit)", "MEM (used)", "MEM (limit)", " ")

			podMetricsList.OrderByHighestMemUsage()
			for _, item := range podMetricsList.Pods {
				fmt.Fprintf(v, format+"\n", item.PodName, item.CPUMillisString(), "-", item.MemoryBytesString(), "-")
			}
			return nil
		})

		<-timerChan
	}
}

func drawTotals(g *gocui.Gui, v *gocui.View) error {
	v.Clear()
	format := "%40s %s\n"
	fmt.Fprintf(v, format, "Uptime:", podMetricsList.Uptime.String())
	fmt.Fprintf(v, format, "Total Nodes in Cluster:", "5")
	fmt.Fprintf(v, format, "Total Memory Available:", "50GB")
	fmt.Fprintf(v, format, "Total CPU Available:", "20")
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("totals", 1, 0, maxX-1, 5); err != nil {
		v.Frame = true
		v.Title = "Kubernetes (minikube)"
		if err != gocui.ErrUnknownView {
			return err
		}

		return drawTotals(g, v)

	}

	if v, err := g.SetView("main", 1, 5, maxX-1, maxY); err != nil {

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

func statsTimer() {
	for {
		timerChan <- 1
		time.Sleep(2 * time.Second)
	}
}
