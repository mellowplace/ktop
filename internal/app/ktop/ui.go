package ktop

import (
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jroimartin/gocui"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

var (
	metricsClose                           = make(chan int, 1)
	metricsReceive                         = make(chan *metrics.PodMetricsList, 1)
	errorChan                              = make(chan error, 1)
	podMetricsList *metrics.PodMetricsList = &metrics.PodMetricsList{}
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

	go collectStats(kubeConfigFile, kubeContextName, namespace, g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

	return nil
}

func collectStats(kubeConfigFile, kubeContextName, namespace string, g *gocui.Gui) {

	go pollPodMetrics(kubeConfigFile, kubeContextName, namespace, metricsClose, metricsReceive, errorChan)

	for {

		select {
		case list, open := <-metricsReceive:
			if open == false {
				return
			}
			podMetricsList = list.DeepCopy()
		case err := <-errorChan:
			g.Update(func(g *gocui.Gui) error {
				return err
			})
			return
		}
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("main")
			if err != nil {
				return err
			}
			v.Clear()

			for _, item := range podMetricsList.Items {
				fmt.Fprintf(v, "%s ", item.GetName())
				labels := item.GetLabels()
				for labelName, label := range labels {
					fmt.Fprintf(v, "%s %s\n", labelName, label)

				}
			}
			return nil
		})
		time.Sleep(2 * time.Second)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("main", -1, -1, maxX, maxY); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}
		spew.Fdump(v, *podMetricsList)
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
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	metricsClose <- 1
	return gocui.ErrQuit
}
