package menu

// BuildMenuItems returns the dropdown items for each top-level menu.
// Actions are set by the App after construction.
type MenuDefs struct {
	OnBriefMode   func()
	OnFullMode    func()
	OnSortName    func()
	OnSortExt     func()
	OnSortSize    func()
	OnSortTime    func()
	OnCopy        func()
	OnMove        func()
	OnMkDir       func()
	OnDelete      func()
	OnQuit        func()
	OnSwapPanels  func()
	OnRefresh     func()
	OnViewFile     func()
	OnEditFile     func()
	OnExportConfig func()
	OnImportConfig func()
	OnQuickPaths   func()
	OnCheckUpdate  func()
}

func LeftMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{
		{Label: "Brief", Key: "", Action: defs.OnBriefMode},
		{Label: "Full", Key: "", Action: defs.OnFullMode},
		{IsSep: true},
		{Label: "Sort by Name", Key: "", Action: defs.OnSortName},
		{Label: "Sort by Ext", Key: "", Action: defs.OnSortExt},
		{Label: "Sort by Size", Key: "", Action: defs.OnSortSize},
		{Label: "Sort by Time", Key: "", Action: defs.OnSortTime},
	}
}

func FileMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{
		{Label: "View", Key: "F3", Action: defs.OnViewFile},
		{Label: "Edit", Key: "F4", Action: defs.OnEditFile},
		{Label: "Copy", Key: "F5", Action: defs.OnCopy},
		{Label: "Move", Key: "F6", Action: defs.OnMove},
		{Label: "MkDir", Key: "F7", Action: defs.OnMkDir},
		{Label: "Delete", Key: "F8", Action: defs.OnDelete},
		{IsSep: true},
		{Label: "Quit", Key: "F10", Action: defs.OnQuit},
	}
}

func CommandsMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{
		{Label: "Swap panels", Key: "", Action: defs.OnSwapPanels},
		{Label: "Refresh", Key: "Ctrl+R", Action: defs.OnRefresh},
		{IsSep: true},
		{Label: "Quick paths", Key: "Ctrl+N", Action: defs.OnQuickPaths},
		{IsSep: true},
		{Label: "Export config", Key: "", Action: defs.OnExportConfig},
		{Label: "Import config", Key: "", Action: defs.OnImportConfig},
		{IsSep: true},
		{Label: "Check for Updates", Key: "", Action: defs.OnCheckUpdate},
	}
}

func OptionsMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{}
}

func RightMenuItems(defs *MenuDefs) []MenuItem {
	return LeftMenuItems(defs)
}
