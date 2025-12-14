package main

import (
	"fmt"
	"time"
)

// ====================== Funciones que generan o son hilos ===========================

// cada uno de estos worker gestiona 1 coche en su fase correspondiente
func worker(g *Garage, entrys [3]chan *Car, exits [3]chan *Car, events chan<- Event, phase int, stop <-chan struct{}) {
	var c *Car
	for {
		c = getCar(entrys, stop)
		if c == nil {
			return
		}
		g.updatePhase(c.id, phase)

		genEvent(events, c, "entra")
		time.Sleep(c.duration)
		genEvent(events, c, "sale")
		if phase == DELIVERYPHASE {
			time.Sleep(c.duration)
			g.delCar(c.id)
			// Liberar una plaza (podrá entrar otro Car en fase 1)
			g.freeSlots <- struct{}{}

			// Este Car ha terminado TODO el ciclo
			g.wg.Done()
		} else {
			sendCar(exits, c)
		}

	}
}

// funcion que genera workers para cada fase
func startPhase(g *Garage, nWorkers int, entrys [3]chan *Car, exits [3]chan *Car, events chan<- Event, phase int, stop <-chan struct{}) {
	for i := 0; i < nWorkers; i++ {
		go worker(g, entrys, exits, events, phase, stop)
	}
}

// ===================== Logger =====================

// este hilo es el único que tiene permitido escribir en stdout
func logger(events <-chan Event) {
	for e := range events {
		secs := e.elapsed.Seconds()
		fmt.Printf("%-9.2f %-9d %-10s %-6d %-8s\n",
			secs,
			e.car,
			e.issue,
			e.phase,
			e.status,
		)

	}
}
