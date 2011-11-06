package main

import (
	"os"
)

type GarboAnt struct {
	visited map[Location]bool
//	curDir Direction
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		visited: make(map[Location]bool),
	}
	return me
}

//DoTurn is where you should do your bot's actual work.
func (me *GarboAnt) DoTurn(s *State) os.Error {
	dirs := []Direction{North, East, South, West}
	for loc, ant := range s.Map.Ants {
		if ant != MY_ANT {
			continue
		}

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
	}
	
	//returning an error will halt the whole program!
	return nil
}
