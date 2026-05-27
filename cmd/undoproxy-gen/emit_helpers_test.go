package main

import "testing"

func TestIndexParamName_avoidsReceiverShadow(t *testing.T) {
	if got := indexParamName("i"); got != "idx" {
		t.Fatalf("got %q want idx", got)
	}
	if got := indexParamName("p"); got != "i" {
		t.Fatalf("got %q want i", got)
	}
}

func TestTruncateLenParamName_avoidsReceiverShadow(t *testing.T) {
	if got := truncateLenParamName("n"); got != "newLen" {
		t.Fatalf("got %q want newLen", got)
	}
	if got := truncateLenParamName("p"); got != "n" {
		t.Fatalf("got %q want n", got)
	}
}
