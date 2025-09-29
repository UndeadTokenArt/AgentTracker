package store

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"agentTracker/internal/models"
)

type Store struct {
	mu     sync.RWMutex
	groups map[string]*models.Group
}

func New() *Store {
	rand.Seed(time.Now().UnixNano())
	return &Store{groups: make(map[string]*models.Group)}
}

func randomCode() string {
	letters := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")
	b := make([]rune, 5)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (s *Store) CreateOrGetGroup(code, uid string) *models.Group {
	s.mu.Lock()
	defer s.mu.Unlock()
	if code == "" {
		code = randomCode()
	}
	g, ok := s.groups[code]
	if !ok {
		g = &models.Group{Code: code, CreatedAt: time.Now(), DMUID: uid, Round: 1}
		s.groups[code] = g
	}
	if g.DMUID == "" {
		g.DMUID = uid
	}
	return g
}

func (s *Store) GetGroup(code string) (*models.Group, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[code]
	return g, ok
}

func (s *Store) AddPlayer(code, uid, name string, initiative, bonus int) (*models.Group, models.Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[code]
	if !ok {
		return nil, models.Entity{}, errors.New("group not found")
	}
	if initiative < 0 {
		initiative = 0
	}
	e := models.Entity{ID: uuid.NewString(), Name: name, Type: models.Player, Initiative: initiative, Bonus: bonus, OwnerUID: uid}
	g.Entities = append(g.Entities, e)
	g.SortOrder()
	return g, e, nil
}

func (s *Store) AddPlayerWithRoll(code, uid, name string, bonus int) (*models.Group, models.Entity, error) {
	roll := models.RollD20()
	return s.AddPlayer(code, uid, name, roll+bonus, bonus)
}

func (s *Store) AddMonster(code, uid, name string, hp, bonus, initiative int) (*models.Group, models.Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[code]
	if !ok {
		return nil, models.Entity{}, errors.New("group not found")
	}
	if g.DMUID != uid {
		return nil, models.Entity{}, errors.New("not dm")
	}
	if initiative < 0 {
		initiative = 0
	}
	e := models.Entity{ID: uuid.NewString(), Name: name, Type: models.Monster, Initiative: initiative, Bonus: bonus, HP: hp, MaxHP: hp}
	g.Entities = append(g.Entities, e)
	g.SortOrder()
	return g, e, nil
}

func (s *Store) DamageMonster(code, uid, entityID string, delta int) (*models.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[code]
	if !ok {
		return nil, errors.New("group not found")
	}
	if g.DMUID != uid {
		return nil, errors.New("not dm")
	}
	for i := range g.Entities {
		if g.Entities[i].ID == entityID && g.Entities[i].Type == models.Monster {
			g.Entities[i].HP -= delta
			if g.Entities[i].HP < 0 {
				g.Entities[i].HP = 0
			}
			return g, nil
		}
	}
	return nil, errors.New("entity not found")
}

func (s *Store) Reorder(code, uid string, order []string) (*models.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[code]
	if !ok {
		return nil, errors.New("group not found")
	}
	if g.DMUID != uid {
		return nil, errors.New("not dm")
	}
	idToEntity := make(map[string]models.Entity)
	for _, e := range g.Entities {
		idToEntity[e.ID] = e
	}
	newList := make([]models.Entity, 0, len(g.Entities))
	for _, id := range order {
		if e, ok := idToEntity[id]; ok {
			newList = append(newList, e)
			delete(idToEntity, id)
		}
	}
	// append any missing
	for _, e := range idToEntity {
		newList = append(newList, e)
	}
	g.Entities = newList
	return g, nil
}

func (s *Store) NextTurn(code string) (*models.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[code]
	if !ok {
		return nil, errors.New("group not found")
	}
	g.NextTurn()
	return g, nil
}
