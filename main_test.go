package main

import "testing"

func TestIsUsableUsername(t *testing.T) {
	usable, err := IsUsableUsername("z")
	if err != nil {
		t.Fatal(err)
	}
	if usable != false {
		t.Errorf("Expected \"false\" got \"%t\"", usable)
		t.Fail()
	}
	usable, err = IsUsableUsername("asdasdasdasdasdaasasd")
	if err != nil {
		t.Fatal(err)
	}
	if usable == false {
		t.Errorf("Expected \"true\" got \"%t\"", usable)
		t.Fail()
	}
}
