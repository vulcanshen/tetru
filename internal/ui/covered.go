package ui

import (
	"reflect"

	terminu "github.com/vulcanshen/terminu-go"
)

// covered says whether a popup is open over the screen: the game stops
// under one (the Space menu its Space opens, ?, a confirm), and goes on as
// it closes.
//
// TODO(tgo): terminu-go has no way to ask whether a popup is open; until it
// does, this reads its stack of popups. A toast is not in it: toasts are not
// popups (tdp F1). TestTheGameStopsUnderAPopup holds it to what the lib does.
func covered(a *terminu.App) bool {
	v := reflect.ValueOf(a).Elem().FieldByName("stack")
	return v.IsValid() && v.Len() > 0
}
