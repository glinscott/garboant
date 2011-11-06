package main

import (
	"os"
)

type GarboAnt struct {
	visible map[Location]float32
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		visible: make(map[Location]float32),
	}
	return me
}

//DoTurn is where you should do your bot's actual work.
func (me *GarboAnt) DoTurn(s *State) os.Error {
	myAnts := []Location{}
	for loc, ant := range s.Map.Ants {
		if ant != MY_ANT {
			continue
		}
		myAnts = append(myAnts, loc);
	}
	
	// Mark the spots as visible
	for _, loc := range myAnts {
		s.Map.DoInRad(loc, s.ViewRadius2, func(row, col int) {
			me.visible[s.Map.FromRowCol(row, col)] = 0.0;
		});
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
