package main

import (
	"os"
	"math"
	"container/list"
)

type GarboAnt struct {
	visible map[Location]float64
	knownFood map[Location]Location
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		visible: make(map[Location]float64),
		knownFood: make(map[Location]Location),
	}
	return me
}

func (me *GarboAnt) FindPath(s *State, source Location, target Location) (Direction, bool) {
	type Node struct {
		current Location;
		startDir Direction;
	}
	
	if (source == target) {
		return North, false;
	}
	
	// BFS
	visited := make(map[Location]bool)
	queue := new(list.List)
	
	nextStep := func(current Location, oldNode *Node) {
		dirs := []Direction{North, East, South, West}
		for _, i := range dirs {
			loc := s.Map.Move(current, dirs[i])
			if (s.Map.SafeDestination(loc)) {
				startDir := dirs[i];
				if oldNode != nil {
					startDir = oldNode.startDir
				}
				queue.PushBack(&Node{current: source, startDir: startDir})
			}
		}		
	}

	nextStep(source, nil);
	
	for queue.Len() != 0 {
		curEl := queue.Front()
		queue.Remove(curEl)
		current := curEl.Value.(Node)
		if current.current == target {
			return current.startDir, true
		}
		if visited[current.current] {
			continue;
		}
		nextStep(current.current, &current)
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
	
	myAnts := []AntState{}
	for loc, ant := range s.Map.Ants {
		if ant != MY_ANT {
			continue
		}
		myAnts = append(myAnts, AntState{ loc : loc, })
	}
	
	// Mark the spots we can see as visible, check for food
	for _, ant := range myAnts {
		s.Map.DoInRad(ant.loc, s.ViewRadius2, func(row, col int) {
			loc := s.Map.FromRowCol(row, col)
			me.visible[loc] = 1.0
			
			if s.Map.Food[loc] {
				me.knownFood[loc] = ant.loc
				ant.closestFood = loc
				ant.huntingFood = true
			}
		});
	}

	// Reduce the visibility
	for row := 0; row < s.Map.Rows; row++ {
		for col := 0; col < s.Map.Cols; col++ {
			loc := s.Map.FromRowCol(row, col)
			me.visible[loc] = math.Fmax(0, me.visible[loc] - 0.01)
		}
	}
	
	// Priorities:
	// 1. Get food
	// 2. Explore
	
	for _, ant := range myAnts {
		if ant.huntingFood {
			// Move towards the food
			targetDir, valid := me.FindPath(s, ant.loc, ant.closestFood)
			if (valid) {
				s.IssueOrderLoc(ant.loc, targetDir)
			}
		} else {
			// Explore
			
		}
	}
	
	
/*	
		dirs := []Direction{North, East, South, West}
		for _, i := range dirs {
			loc2 := s.Map.Move(loc, dirs[i])
			if me.visited[loc2] {
				continue
		  }
			if s.Map.SafeDestination(loc2) {
				me.visited[loc2] = true
				s.IssueOrderLoc(loc, dirs[i])
				break
			}
		}
	}*/
	
	//returning an error will halt the whole program!
	return nil
}
