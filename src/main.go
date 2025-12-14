package main

import (
	"fmt"
	"time"
)

// ===================== Función main =====================

func main() {
	var g *Garage
	var noExits [3]chan *Car // para fase 4 que no tiene canal de salida
	numCars := 20            // N coches totales
	numSlots := 10           // numPlazas
	numMechs := 4            // numMecanicos

	numWorkersDoc := numSlots
	numWorkersRep := numMechs // reparación limitada por mecánicos
	numWorkersClean := numSlots
	numWorkersDeliber := numSlots

	// Canales entre fases
	docChans := initPhaseChans()
	repChans := initPhaseChans()
	cleanChans := initPhaseChans()
	deliverChans := initPhaseChans()
	// Canal para events hacia el logger
	events := make(chan Event)

	// canales para terminar a las goroutines
	stopDoc := make(chan struct{})
	stopRep := make(chan struct{})
	stopClean := make(chan struct{})
	stopDeliver := make(chan struct{})

	g = newGarage(numSlots)
	carspool := genCars(numCars)

	fmt.Printf("%-8s %-8s %-10s %-6s %-8s\n",
		"Tiempo[s]", "Coche[id]", "Incidencia", "Fase", "Estado")
	fmt.Printf("-------------------------------------------------\n")
	// Lanzar logger (único que imprime por stdout)
	go logger(events)

	// Lanzar workers de cada fase

	// Fase de documentacion:
	startPhase(g, numWorkersDoc, docChans, repChans, events, DOCPHASE, stopDoc)

	// Fase de reparacion
	startPhase(g, numWorkersRep, repChans, cleanChans, events, REPAIRPHASE, stopRep)

	// Fase de limpieza
	startPhase(g, numWorkersClean, cleanChans, deliverChans, events, CLEANPHASE, stopClean)

	// Fase de entrega
	startPhase(g, numWorkersDeliber, deliverChans, noExits, events, DELIVERYPHASE, stopDeliver)

	// Generación de Cars (productor)
	for i := 0; i < numCars; i++ {
		// Esperar a tener una plaza libre
		<-g.freeSlots

		c := getCarFromQ(carspool)
		c.start = time.Now()
		// Registrar el Car en el mapa global
		g.signInCar(c)

		// Este Car hará todo el ciclo -> lo contamos en el WaitGroup
		g.wg.Add(1)

		// Entra en la primera fase (documentación)
		sendCar(docChans, c)
	}

	// Esperar a que TODOS los Cars terminen la fase de entrega
	g.wg.Wait()
	freeThreats(stopDoc, stopRep, stopClean, stopDeliver)
	closeChans(docChans)
	closeChans(repChans)
	closeChans(cleanChans)
	closeChans(deliverChans)
	close(events)
	close(g.freeSlots)
	fmt.Println("Simulación completada.")
}
