package password

import "testing"

func TestValidCertTitle(t *testing.T) {
	if !ValidCertTitle("ab1") {
		t.Fatal("expected valid")
	}
	if ValidCertTitle("ab") {
		t.Fatal("too short")
	}
	if ValidCertTitle("bad space") {
		t.Fatal("spaces invalid")
	}
}

func TestGenerateCertKeyPass(t *testing.T) {
	pw, err := GenerateCertKeyPass()
	if err != nil {
		t.Fatal(err)
	}
	if len(pw) < 8 {
		t.Fatalf("short pass: %q", pw)
	}
}
