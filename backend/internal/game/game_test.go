package game

import "testing"

func TestAllCombinations(t *testing.T) {
	counts := map[int64]int{}
	for a := 0; a < 10; a++ {
		for b := 0; b < 10; b++ {
			for c := 0; c < 10; c++ {
				counts[Payout([3]int{a, b, c}, 1)]++
			}
		}
	}
	for payout, count := range map[int64]int{100: 1, 20: 9, 2: 270, 0: 720} {
		if counts[payout] != count {
			t.Fatalf("payout %d: got %d combinations, want %d", payout, counts[payout], count)
		}
	}
	for _, tc := range []struct {
		n    [3]int
		want int64
	}{
		{[3]int{7, 7, 7}, 10000}, {[3]int{0, 0, 0}, 2000}, {[3]int{1, 2, 1}, 200}, {[3]int{1, 2, 3}, 0},
	} {
		if got := Payout(tc.n, 100); got != tc.want {
			t.Fatalf("%v: got %d, want %d", tc.n, got, tc.want)
		}
	}
}

func TestCryptoDrawRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		n, err := CryptoDraw()
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range n {
			if v < 0 || v > 9 {
				t.Fatalf("out of range: %v", n)
			}
		}
	}
}
