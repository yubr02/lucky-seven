package game

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"regexp"
	"time"
)

var (
	ErrInvalid      = errors.New("invalid bet or idempotency key")
	ErrInsufficient = errors.New("insufficient balance")
	ErrConflict     = errors.New("idempotency key already used with another bet")
)

type Result struct {
	ID        int64     `json:"id"`
	Numbers   [3]int    `json:"numbers"`
	Bet       int64     `json:"bet"`
	Payout    int64     `json:"payout"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type Draw func() ([3]int, error)

func CryptoDraw() ([3]int, error) {
	var numbers [3]int
	for i := range numbers {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return [3]int{}, err
		}
		numbers[i] = int(n.Int64())
	}
	return numbers, nil
}

func Payout(n [3]int, bet int64) int64 {
	if n == [3]int{7, 7, 7} {
		return 100 * bet
	}
	if n[0] == n[1] && n[1] == n[2] {
		return 20 * bet
	}
	if n[0] == n[1] || n[0] == n[2] || n[1] == n[2] {
		return 2 * bet
	}
	return 0
}

type Repository interface {
	Play(context.Context, string, string, int64, Draw) (Result, error)
	Balance(context.Context, string) (int64, error)
	History(context.Context, string, int64) ([]Result, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

var validKey = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func (s *Service) Play(ctx context.Context, player, key string, bet int64) (Result, error) {
	if bet < 1 || bet > 100 || !validKey.MatchString(key) {
		return Result{}, ErrInvalid
	}
	return s.repo.Play(ctx, player, key, bet, CryptoDraw)
}

func (s *Service) Balance(ctx context.Context, player string) (int64, error) {
	return s.repo.Balance(ctx, player)
}

func (s *Service) History(ctx context.Context, player string, before int64) ([]Result, error) {
	return s.repo.History(ctx, player, before)
}
