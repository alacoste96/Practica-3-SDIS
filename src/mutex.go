package main

// ----- Métodos protegidos por RWMutex -----

// La zona crítica es el map de coches (atributo de garage)

// funcion que da de alta un coche en el taller(crea un coche en el map)
func (g *Garage) signInCar(c *Car) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cars[c.id] = c
}

// funcion que actualiza la fase en la que se encuentra un coche (consulta el map)
func (g *Garage) updatePhase(id int, phase int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if c, ok := g.cars[id]; ok {
		c.curphase = phase
	}
}

// funcion que elimina un coche del map
func (g *Garage) delCar(id int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.cars, id)
}
