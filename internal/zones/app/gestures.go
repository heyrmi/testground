package app

import "github.com/heyrmi/testground/internal/challenge"

func dragAndDrop() challenge.Challenge {
	return challenge.Challenge{
		ID:       "drag-and-drop",
		Title:    "Two kinds of dragging, and they are nothing alike",
		URL:      "/app/drag-and-drop",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "J. Interactions & Gestures",
		Summary: "Parcels moved into a drop zone with the native HTML5 drag events, a list " +
			"reordered the same way, and a handle that moves only for a pointer sequence " +
			"with no drag events involved at all.",
		WhyHard: "Native drag and drop is not mouse movement. It is a separate event family the " +
			"browser raises on the operating system's behalf, and moving a mouse across the " +
			"screen does not produce it -- press, move and release the mouse across the drop " +
			"zone and the parcel simply stays put, with nothing raised to say why. That is " +
			"also why several drivers cannot do this at all. The pointer handle has the " +
			"opposite requirement: it needs a real " +
			"press, at least one intervening move and a release, because there is no drop " +
			"target to jump to and a single move to the destination lands nowhere.",
		Hint: "The two are not symmetric, and it is worth knowing which way round. A proper " +
			"drag helper drives real input and turns on the browser's drag machinery, so it " +
			"satisfies both the native parcels and the pointer handle. Pressing and moving " +
			"the mouse yourself only satisfies the handle: the parcels stay where they are " +
			"and nothing reports an error. So reach for the helper first, fall back to " +
			"dispatching drag events with a dataTransfer, and remember that raw coordinates " +
			"do not scroll anything into view.",
		Tags:     []string{"drag", "drop", "pointer", "sortable"},
		Concepts: []string{"HTML5 drag is not mouse movement", "dataTransfer", "pointer sequences need intermediate moves", "reordering by drop target"},
		Selectors: []challenge.Selector{
			{TestID: "parcel-source", Note: "Holds the parcels still to be moved"},
			{TestID: "parcel", Note: "A draggable parcel; narrow by data-name. They leave as they are delivered"},
			{TestID: "dropzone", Note: "Accepts parcels dropped on it"},
			{TestID: "delivered", Transient: true, Note: "One per parcel that arrived"},
			{TestID: "delivered-count", Note: "How many made it"},
			{TestID: "sortable", Note: "The reorderable list"},
			{TestID: "sortable-item", Note: "One entry; dropping one onto another moves it there"},
			{TestID: "sortable-order", Note: "The current order, in text"},
			{TestID: "rail", Note: "Press and drag along this; no drag events, only pointer ones"},
			{TestID: "handle", Note: "Follows the pointer while a button is held"},
			{TestID: "handle-position", Note: "Zero to a hundred"},
		},
		Stability: challenge.Stable,
	}
}

func pointerMenus() challenge.Challenge {
	return challenge.Challenge{
		ID:       "pointer-menus",
		Title:    "Menus that need the pointer to stay where it is",
		URL:      "/app/pointer-menus",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "J. Interactions & Gestures",
		Summary: "A menu that exists only while hovered, with a submenu one level further in, " +
			"beside a target that answers right-click with its own menu, counts double " +
			"clicks separately from single ones, and recognises a half-second hold.",
		WhyHard: "The hover menu is not in the DOM until the pointer is inside its group, and it " +
			"leaves the instant the pointer does -- so anything that moves the pointer " +
			"elsewhere between finding an item and clicking it finds nothing. Reaching the " +
			"submenu means travelling through the parent without straying outside. The " +
			"gesture target is the other trap: a double click also raises two single " +
			"clicks, so the counters never agree and a test asserting one click happened " +
			"after a double click is asserting something false. The long press is a hold " +
			"rather than a click, and no click helper produces one.",
		Hint: "Hover the trigger and then move onto the menu itself rather than clicking " +
			"blindly; a framework's hover will keep the pointer where it put it. For the " +
			"submenu, hover the parent item first and let it open. Expect a double click to " +
			"increment both counters. For the hold, press, wait, then release -- three " +
			"separate steps, because a click is all three at once and far too quickly.",
		Tags:     []string{"hover", "context-menu", "double-click", "long-press", "submenu"},
		Concepts: []string{"hover-only elements", "traversing into a submenu", "a double click is also two clicks", "press and hold is not a click"},
		Selectors: []challenge.Selector{
			{TestID: "hover-root", Note: "The group; the menu lives only while the pointer is inside it"},
			{TestID: "hover-trigger", Role: "button", Note: "Hovering this opens the menu"},
			{TestID: "hover-menu", Transient: true, Note: "Present only while hovered"},
			{TestID: "menu-open", Role: "button", Transient: true, Note: "Inside the hover menu"},
			{TestID: "menu-more", Role: "button", Transient: true, Note: "Hovering this opens the submenu"},
			{TestID: "submenu", Transient: true, Note: "One level further in"},
			{TestID: "menu-archive", Role: "button", Transient: true, Note: "Inside the submenu"},
			{TestID: "menu-choice", Note: "What was chosen, which outlives every menu here"},
			{TestID: "gesture-target", Note: "Answers right-click, double-click and a held press"},
			{TestID: "context-menu", Transient: true, Note: "Replaces the browser's own"},
			{TestID: "context-rename", Role: "button", Transient: true, Note: "Inside the context menu"},
			{TestID: "single-clicks", Note: "Counts every click, including the two inside a double"},
			{TestID: "double-clicks", Note: "Counts only double clicks"},
			{TestID: "long-presses", Note: "Counts holds of at least half a second"},
		},
		Stability: challenge.Stable,
	}
}
