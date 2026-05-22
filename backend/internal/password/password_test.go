package password

import "testing"

func TestValidVPN(t *testing.T) {
	if !ValidVPN("Secret123") {
		t.Fatal("valid VPN password rejected")
	}
	for _, pw := range []string{"short1", "onlyletters", "12345678", ""} {
		if ValidVPN(pw) {
			t.Errorf("ValidVPN(%q) should be false", pw)
		}
	}
}

func TestValidPanel(t *testing.T) {
	if !ValidPanel("Manager1") {
		t.Fatal("valid panel password rejected")
	}
	if ValidPanel("short") || ValidPanel("12345678") {
		t.Fatal("invalid panel passwords accepted")
	}
}

func TestHashAndCheck(t *testing.T) {
	hash, err := Hash("TestPass1")
	if err != nil {
		t.Fatal(err)
	}
	if !Check(hash, "TestPass1") {
		t.Fatal("Check should succeed")
	}
	if Check(hash, "wrong") {
		t.Fatal("Check should fail for wrong password")
	}
}
