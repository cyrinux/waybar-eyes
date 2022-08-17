package eyes

import (
	"strings"
	"testing"
)

func Test_ClassShouldBeOK(t *testing.T) {
	eyes := Eyes{}
	want := "ok"

	eyes.PrepareWaybarOutput()
	got := eyes.Class
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func Test_ClassShouldBeCritical(t *testing.T) {
	eyes := Eyes{Count: MaxEyes}
	want := "critical"

	eyes.PrepareWaybarOutput()
	got := eyes.Class
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func Test_TextShouldHaveEnoughEyes(t *testing.T) {
	eyes := Eyes{Count: MaxEyes}
	want := strings.Repeat(EYE, MaxEyes)

	eyes.PrepareWaybarOutput()
	got := eyes.Text
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	eyes.Count = 0
	want = ""
	eyes.PrepareWaybarOutput()
	got = eyes.Text
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

}
