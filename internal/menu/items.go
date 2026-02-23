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
	OnSymlink      func()
	OnChmod        func()
}

func LeftMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{
		{Label: "Brief", Key: "", Action: defs.OnBriefMode, HotKey: 'B'},
		{Label: "Full", Key: "", Action: defs.OnFullMode, HotKey: 'U'},
		{IsSep: true},
		{Label: "Sort by Name", Key: "", Action: defs.OnSortName, HotKey: 'N'},
		{Label: "Sort by Ext", Key: "", Action: defs.OnSortExt, HotKey: 'E'},
		{Label: "Sort by Size", Key: "", Action: defs.OnSortSize, HotKey: 'S'},
		{Label: "Sort by Time", Key: "", Action: defs.OnSortTime, HotKey: 'T'},
	}
}

func FileMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{
		{Label: "View", Key: "F3", Action: defs.OnViewFile, HotKey: 'V'},
		{Label: "Edit", Key: "F4", Action: defs.OnEditFile, HotKey: 'E'},
		{Label: "Copy", Key: "F5", Action: defs.OnCopy, HotKey: 'C'},
		{Label: "Move", Key: "F6", Action: defs.OnMove, HotKey: 'M'},
		{Label: "MkDir", Key: "F7", Action: defs.OnMkDir, HotKey: 'K'},
		{Label: "Delete", Key: "F8", Action: defs.OnDelete, HotKey: 'D'},
		{Label: "Symlink", Key: "", Action: defs.OnSymlink, HotKey: 'S'},
		{Label: "Attribútum", Key: "", Action: defs.OnChmod, HotKey: 'A'},
		{IsSep: true},
		{Label: "Quit", Key: "F10", Action: defs.OnQuit, HotKey: 'Q'},
	}
}

func CommandsMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{
		{Label: "Swap panels", Key: "", Action: defs.OnSwapPanels, HotKey: 'S'},
		{Label: "Refresh", Key: "Ctrl+R", Action: defs.OnRefresh, HotKey: 'R'},
		{IsSep: true},
		{Label: "Quick paths", Key: "Ctrl+N", Action: defs.OnQuickPaths, HotKey: 'Q'},
		{IsSep: true},
		{Label: "Export config", Key: "", Action: defs.OnExportConfig, HotKey: 'E'},
		{Label: "Import config", Key: "", Action: defs.OnImportConfig, HotKey: 'I'},
		{IsSep: true},
		{Label: "Check for Updates", Key: "", Action: defs.OnCheckUpdate, HotKey: 'U'},
	}
}

func OptionsMenuItems(defs *MenuDefs) []MenuItem {
	return []MenuItem{}
}

func RightMenuItems(defs *MenuDefs) []MenuItem {
	return LeftMenuItems(defs)
}
