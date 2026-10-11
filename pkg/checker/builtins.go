package checker

import "strings"

func IsBuiltinUnit(name string) bool {
	switch strings.ToLower(name) {
	case "std", "net", "fs", "crypto", "sys", "cmd", "path", "local", "json", "math":
		return true
	}
	return false
}

func (c *Checker) initBuiltinUnits() {
	netUnit := &Symbol{
		Name:       "Net",
		Kind:       SymUnit,
		Directives: map[string]bool{"set": true, "header": true, "on": true},
		Methods: map[string]*Symbol{
			"Fetch": {Name: "Fetch", Kind: SymFunc, Type: "any"},
			"fetch": {Name: "fetch", Kind: SymFunc, Type: "any"},
			"Clone": {Name: "Clone", Kind: SymFunc, Type: "bool"},
			"clone": {Name: "clone", Kind: SymFunc, Type: "bool"},
			"Get":   {Name: "Get", Kind: SymFunc, Type: "string"},
			"get":   {Name: "get", Kind: SymFunc, Type: "string"},
		},
	}
	fsUnit := &Symbol{
		Name:       "Fs",
		Kind:       SymUnit,
		Directives: map[string]bool{"set": true},
		Methods: map[string]*Symbol{
			"Exists":    {Name: "Exists", Kind: SymFunc, Type: "bool"},
			"exists":    {Name: "exists", Kind: SymFunc, Type: "bool"},
			"IsDir":     {Name: "IsDir", Kind: SymFunc, Type: "bool"},
			"isDir":     {Name: "isDir", Kind: SymFunc, Type: "bool"},
			"IsFile":    {Name: "IsFile", Kind: SymFunc, Type: "bool"},
			"isFile":    {Name: "isFile", Kind: SymFunc, Type: "bool"},
			"Read":      {Name: "Read", Kind: SymFunc, Type: "string"},
			"read":      {Name: "read", Kind: SymFunc, Type: "string"},
			"Size":      {Name: "Size", Kind: SymFunc, Type: "number"},
			"size":      {Name: "size", Kind: SymFunc, Type: "number"},
			"Remove":    {Name: "Remove", Kind: SymFunc, Type: "void"},
			"remove":    {Name: "remove", Kind: SymFunc, Type: "void"},
			"Move":      {Name: "Move", Kind: SymFunc, Type: "void"},
			"move":      {Name: "move", Kind: SymFunc, Type: "void"},
			"Copy":      {Name: "Copy", Kind: SymFunc, Type: "void"},
			"copy":      {Name: "copy", Kind: SymFunc, Type: "void"},
			"Write":     {Name: "Write", Kind: SymFunc, Type: "bool"},
			"write":     {Name: "write", Kind: SymFunc, Type: "bool"},
			"EnsureDir": {Name: "EnsureDir", Kind: SymFunc, Type: "void"},
			"ensureDir": {Name: "ensureDir", Kind: SymFunc, Type: "void"},
			"ensuredir": {Name: "ensuredir", Kind: SymFunc, Type: "void"},
		},
	}
	stdUnit := &Symbol{
		Name: "std",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"log":            {Name: "log", Kind: SymFunc, Type: "void"},
			"Log":            {Name: "Log", Kind: SymFunc, Type: "void"},
			"println":        {Name: "println", Kind: SymFunc, Type: "void"},
			"Println":        {Name: "Println", Kind: SymFunc, Type: "void"},
			"print":          {Name: "print", Kind: SymFunc, Type: "void"},
			"Print":          {Name: "Print", Kind: SymFunc, Type: "void"},
			"info":           {Name: "info", Kind: SymFunc, Type: "void"},
			"Info":           {Name: "Info", Kind: SymFunc, Type: "void"},
			"success":        {Name: "success", Kind: SymFunc, Type: "void"},
			"Success":        {Name: "Success", Kind: SymFunc, Type: "void"},
			"warn":           {Name: "warn", Kind: SymFunc, Type: "void"},
			"Warn":           {Name: "Warn", Kind: SymFunc, Type: "void"},
			"error":          {Name: "error", Kind: SymFunc, Type: "void"},
			"Error":          {Name: "Error", Kind: SymFunc, Type: "void"},
		},
	}
	sysUnit := &Symbol{
		Name: "Sys",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"GetEnv": {Name: "GetEnv", Kind: SymFunc, Type: "string"},
			"getEnv": {Name: "getEnv", Kind: SymFunc, Type: "string"},
			"Os":     {Name: "Os", Kind: SymFunc, Type: "string"},
			"os":     {Name: "os", Kind: SymFunc, Type: "string"},
			"Arch":   {Name: "Arch", Kind: SymFunc, Type: "string"},
			"arch":   {Name: "arch", Kind: SymFunc, Type: "string"},
		},
	}
	cmdUnit := &Symbol{
		Name: "Cmd",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Exec": {Name: "Exec", Kind: SymFunc, Type: "string"},
			"exec": {Name: "exec", Kind: SymFunc, Type: "string"},
		},
	}
	cryptoUnit := &Symbol{
		Name: "Crypto",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"GetRandomBytes": {Name: "GetRandomBytes", Kind: SymFunc, Type: "string"},
			"getRandomBytes": {Name: "getRandomBytes", Kind: SymFunc, Type: "string"},
			"RandomBytes":    {Name: "RandomBytes", Kind: SymFunc, Type: "string"},
			"randomBytes":    {Name: "randomBytes", Kind: SymFunc, Type: "string"},
			"RandomHex":      {Name: "RandomHex", Kind: SymFunc, Type: "string"},
			"randomHex":      {Name: "randomHex", Kind: SymFunc, Type: "string"},
			"RandomBase64":   {Name: "RandomBase64", Kind: SymFunc, Type: "string"},
			"randomBase64":   {Name: "randomBase64", Kind: SymFunc, Type: "string"},
			"Sha256":         {Name: "Sha256", Kind: SymFunc, Type: "string"},
			"sha256":         {Name: "sha256", Kind: SymFunc, Type: "string"},
			"Uuid":           {Name: "Uuid", Kind: SymFunc, Type: "string"},
			"uuid":           {Name: "uuid", Kind: SymFunc, Type: "string"},
		},
	}

	localUnit := &Symbol{
		Name: "Local",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Keep":       {Name: "Keep", Kind: SymFunc, Type: "void"},
			"Drop":       {Name: "Drop", Kind: SymFunc, Type: "void"},
			"UseVariant": {Name: "UseVariant", Kind: SymFunc, Type: "void"},
		},
	}

	pathUnit := &Symbol{
		Name: "Path",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Join": {Name: "Join", Kind: SymFunc, Type: "string"},
			"join": {Name: "join", Kind: SymFunc, Type: "string"},
			"Dir":  {Name: "Dir", Kind: SymFunc, Type: "string"},
			"dir":  {Name: "dir", Kind: SymFunc, Type: "string"},
			"Base": {Name: "Base", Kind: SymFunc, Type: "string"},
			"base": {Name: "base", Kind: SymFunc, Type: "string"},
			"Ext":  {Name: "Ext", Kind: SymFunc, Type: "string"},
			"ext":  {Name: "ext", Kind: SymFunc, Type: "string"},
		},
	}
	jsonUnit := &Symbol{
		Name: "Json",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Parse":     {Name: "Parse", Kind: SymFunc, Type: "any"},
			"parse":     {Name: "parse", Kind: SymFunc, Type: "any"},
			"Stringify": {Name: "Stringify", Kind: SymFunc, Type: "string"},
			"stringify": {Name: "stringify", Kind: SymFunc, Type: "string"},
		},
	}
	mathUnit := &Symbol{
		Name: "Math",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Round":  {Name: "Round", Kind: SymFunc, Type: "number"},
			"round":  {Name: "round", Kind: SymFunc, Type: "number"},
			"Floor":  {Name: "Floor", Kind: SymFunc, Type: "number"},
			"floor":  {Name: "floor", Kind: SymFunc, Type: "number"},
			"Ceil":   {Name: "Ceil", Kind: SymFunc, Type: "number"},
			"ceil":   {Name: "ceil", Kind: SymFunc, Type: "number"},
			"Abs":    {Name: "Abs", Kind: SymFunc, Type: "number"},
			"abs":    {Name: "abs", Kind: SymFunc, Type: "number"},
			"Min":    {Name: "Min", Kind: SymFunc, Type: "number"},
			"min":    {Name: "min", Kind: SymFunc, Type: "number"},
			"Max":    {Name: "Max", Kind: SymFunc, Type: "number"},
			"max":    {Name: "max", Kind: SymFunc, Type: "number"},
			"Random": {Name: "Random", Kind: SymFunc, Type: "number"},
			"random": {Name: "random", Kind: SymFunc, Type: "number"},
		},
	}

	c.availableModules["Net"] = netUnit
	c.availableModules["Fs"] = fsUnit
	c.availableModules["std"] = stdUnit
	c.availableModules["Std"] = stdUnit
	c.availableModules["Sys"] = sysUnit
	c.availableModules["Cmd"] = cmdUnit
	c.availableModules["Crypto"] = cryptoUnit
	c.availableModules["Local"] = localUnit
	c.availableModules["Path"] = pathUnit
	c.availableModules["path"] = pathUnit
	c.availableModules["Json"] = jsonUnit
	c.availableModules["json"] = jsonUnit
	c.availableModules["Math"] = mathUnit
	c.availableModules["math"] = mathUnit

	topLevelFuncs := []string{
		"Drop", "drop", "Keep", "keep",
		"replace", "replaceLine", "remove", "removeLines", "removeBlock",
		"removeUntil", "commentOut", "append", "prepend", "jsonSet", "jsonRemove",
	}
	for _, fn := range topLevelFuncs {
		c.scope.Define(&Symbol{Name: fn, Kind: SymFunc, Type: "any"})
	}
}

func (c *Checker) DefineUnit(name string) {
	for mName, sym := range c.availableModules {
		if strings.EqualFold(mName, name) {
			_ = c.scope.Define(sym)
			return
		}
	}
}

func IsStringMethod(name string) bool {
	switch strings.ToLower(name) {
	case "len", "trim", "trimleft", "trimright", "toupper", "tolower",
		"contains", "startswith", "endswith", "split", "replace", "slice":
		return true
	}
	return false
}

func IsListMethod(name string) bool {
	switch strings.ToLower(name) {
	case "len", "join", "contains", "push", "pop", "reverse", "slice":
		return true
	}
	return false
}

