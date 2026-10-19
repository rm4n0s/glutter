package gestures

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
)

type combiningArenaEntry struct {
	combiner *combiningArenaMember
	member   GestureArenaMember
}

func (e *combiningArenaEntry) Resolve(disposition GestureDisposition) {
	e.combiner.resolve(e.member, disposition)
}

// combiningArenaMember is the one member that a team puts in the arena for all
// its members.
type combiningArenaMember struct {
	owner    *GestureArenaTeam
	members  []GestureArenaMember
	pointer  int
	resolved bool
	winner   GestureArenaMember
	entry    GestureArenaEntry
}

func (c *combiningArenaMember) AcceptGesture(pointer int) {
	if foundation.DebugMode && (c.pointer != pointer || (c.winner == nil && len(c.members) == 0)) {
		panic(errors.New("TeamAcceptInvalid", "the team has no member to accept for the pointer", "pointer", pointer))
	}
	c.close()
	if c.winner == nil {
		if c.owner.Captain != nil {
			c.winner = c.owner.Captain
		} else {
			c.winner = c.members[0]
		}
	}
	for _, member := range c.members {
		if member != c.winner {
			member.RejectGesture(pointer)
		}
	}
	c.winner.AcceptGesture(pointer)
}

func (c *combiningArenaMember) RejectGesture(pointer int) {
	c.close()
	for _, member := range c.members {
		member.RejectGesture(pointer)
	}
}

func (c *combiningArenaMember) close() {
	if foundation.DebugMode && c.resolved {
		panic(errors.New("TeamResolvedTwice", "the team's member was resolved twice", "pointer", c.pointer))
	}
	c.resolved = true
	delete(c.owner.combiners, c.pointer)
}

func (c *combiningArenaMember) add(pointer int, member GestureArenaMember) GestureArenaEntry {
	if foundation.DebugMode && (c.resolved || c.pointer != pointer) {
		panic(errors.New("TeamAddInvalid", "a member cannot join a team's resolved member", "pointer", pointer))
	}
	c.members = append(c.members, member)
	if c.entry == nil {
		c.entry = c.owner.arena.Add(pointer, c)
	}
	return &combiningArenaEntry{c, member}
}

func (c *combiningArenaMember) resolve(member GestureArenaMember, disposition GestureDisposition) {
	if c.resolved {
		return
	}
	switch disposition {
	case GestureAccepted:
		if c.winner == nil {
			if c.owner.Captain != nil {
				c.winner = c.owner.Captain
			} else {
				c.winner = member
			}
		}
		c.entry.Resolve(disposition)
	case GestureRejected:
		for i, m := range c.members {
			if m == member {
				c.members = append(c.members[:i:i], c.members[i+1:]...)
				break
			}
		}
		member.RejectGesture(c.pointer)
		if len(c.members) == 0 {
			c.entry.Resolve(disposition)
		}
	}
}

// GestureArenaTeam combines the members that join it into one member of the
// arena. The members are on the same team: only one of them wins, and the others
// lose with it. The captain, if there is one, wins when the team does.
//
// Flutter: gestures/team.dart GestureArenaTeam
type GestureArenaTeam struct {
	arena     *GestureArenaManager
	combiners map[int]*combiningArenaMember

	// Captain is the member that wins when the team wins; if it is nil, the
	// first member to accept or the first to join wins.
	//
	// Flutter: gestures/team.dart GestureArenaTeam.captain
	Captain GestureArenaMember
}

// NewGestureArenaTeam creates a team that competes in the arena.
//
// Flutter: gestures/team.dart GestureArenaTeam.new
func NewGestureArenaTeam(arena *GestureArenaManager) *GestureArenaTeam {
	return &GestureArenaTeam{arena: arena, combiners: map[int]*combiningArenaMember{}}
}

// Add adds the member to the team's combined member in the arena of the
// pointer.
//
// Flutter: gestures/team.dart GestureArenaTeam.add
func (t *GestureArenaTeam) Add(pointer int, member GestureArenaMember) GestureArenaEntry {
	combiner, ok := t.combiners[pointer]
	if !ok {
		combiner = &combiningArenaMember{owner: t, pointer: pointer}
		t.combiners[pointer] = combiner
	}
	return combiner.add(pointer, member)
}
