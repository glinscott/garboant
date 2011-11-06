package main

import (
	"container/list"
	"log"
	"math"
	"os"
)

type GarboAnt struct {
	visible map[Location]float64
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		visible: make(map[Location]float64),
	}
	return me
}

func (me *GarboAnt) SearchMap(s *State, source Location, isFinalState func(current Location) bool) (Direction, bool) {
	type Node struct {
		current Location
		startDir Direction
	}

	// BFS
	visited := make(map[Location]bool)
	queue := new(list.List)
	
	nextStep := func(current Location, oldNode *Node) {
		dirs := []Direction{North, East, South, West}
		for _, i := range dirs {
			loc := s.Map.Move(current, dirs[i])
			if (s.Map.SafeDestination(loc) && !visited[loc]) {
				startDir := dirs[i];
				if oldNode != nil {
					startDir = oldNode.startDir
				}
				queue.PushBack(&Node{current: loc, startDir: startDir})
			}
		}		
	}

	visited[source] = true
	nextStep(source, nil)

	for queue.Len() != 0 {
		curEl := queue.Front()
		queue.Remove(curEl)
		current := curEl.Value.(*Node)
		if isFinalState(current.current) {
			return current.startDir, true
		}
		if visited[current.current] {
			continue;
		}
		visited[current.current] = true
		nextStep(current.current, current)
	}

	return North, false
}

//DoTurn is where you should do your bot's actual work.
func (me *GarboAnt) DoTurn(s *State) os.Error {
	type AntState struct {
		loc Location
		closestFood Location
		huntingFood bool
	}
	
	myAnts := []*AntState{}
	for loc, ant := range s.Map.Ants {
		if ant != MY_ANT {
			continue
		}
		myAnts = append(myAnts, &AntState{ loc: loc, huntingFood: false,})
	}
	
	// Reduce the visibility
	for row := 0; row < s.Map.Rows; row++ {
		for col := 0; col < s.Map.Cols; col++ {
			loc := s.Map.FromRowCol(row, col)
			me.visible[loc] = math.Fmax(0, me.visible[loc] - 0.01)
		}
	}

	// Mark the spots we can see as visible, check for food
	for _, ant := range myAnts {
		closest := 9999999
		fRow, fCol := s.Map.FromLocation(ant.loc)
		s.Map.DoInRad(ant.loc, s.ViewRadius2, func(row, col int) {
			loc := s.Map.FromRowCol(row, col)
			me.visible[loc] = 1.0
			
			if s.Map.Food[loc] {
				distance := (fRow-row)*(fRow-row)+(fCol-col)*(fCol-col);
				if distance < closest {
					closest = distance
					ant.closestFood = loc
					ant.huntingFood = true
				}
			}
		});
	}
	
	safeMove := func(loc Location, dir Direction) bool {
		target := s.Map.Move(loc, dir)
		if (s.Map.SafeDestination(target)) {
			s.IssueOrderLoc(loc, dir)
			return true
		}
		return false
	}

	// Priorities:
	// 1. Get food
	// 2. Explore
	for _, ant := range myAnts {
		if ant.huntingFood {
			// Move towards the food
			finalState := func(current Location) bool {
				return current == ant.closestFood
			}
			targetDir, valid := me.SearchMap(s, ant.loc, finalState)
			if (valid) {
				safeMove(ant.loc, targetDir)
			}
		} else {
			// Explore
			finalState := func(current Location) bool {
				return me.visible[current] != 1.0
			}
			targetDir, valid := me.SearchMap(s, ant.loc, finalState)
			if (valid) {
				safeMove(ant.loc, targetDir)
			}
		}
	}
	
	log.Println("Finished turn in ms")
	//returning an error will halt the whole program!
	return nil
}
