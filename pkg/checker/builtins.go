package checker

import (
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func IsBuiltinUnit(name string) bool {
	switch strings.ToLower(name) {
	case "std", "net", "fs", "crypto", "sys", "cmd", "path", "local", "json", "math", "date":
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
			"Get":             {Name: "Get", Kind: SymFunc, Type: "string"},
			"get":             {Name: "get", Kind: SymFunc, Type: "string"},
			"isPortBusy":      {Name: "isPortBusy", Kind: SymFunc, Type: "bool"},
			"IsPortBusy":      {Name: "IsPortBusy", Kind: SymFunc, Type: "bool"},
			"isPortFree":      {Name: "isPortFree", Kind: SymFunc, Type: "bool"},
			"IsPortFree":      {Name: "IsPortFree", Kind: SymFunc, Type: "bool"},
			"isPortAvailable": {Name: "isPortAvailable", Kind: SymFunc, Type: "bool"},
			"IsPortAvailable": {Name: "IsPortAvailable", Kind: SymFunc, Type: "bool"},
			"isPortInUse":     {Name: "isPortInUse", Kind: SymFunc, Type: "bool"},
			"IsPortInUse":     {Name: "IsPortInUse", Kind: SymFunc, Type: "bool"},
		},
	}
	fsUnit := &Symbol{
		Name:       "Fs",
		Kind:       SymUnit,
		Directives: map[string]bool{"set": true},
		Methods: map[string]*Symbol{
			"Exists":    {Name: "Exists", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"exists":    {Name: "exists", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"IsDir":     {Name: "IsDir", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"isDir":     {Name: "isDir", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"IsFile":    {Name: "IsFile", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"isFile":    {Name: "isFile", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"Read":      {Name: "Read", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"read":      {Name: "read", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"Size":      {Name: "Size", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"size":      {Name: "size", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"Remove":    {Name: "Remove", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"remove":    {Name: "remove", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"Move":      {Name: "Move", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "src", Type: "string"}, {Name: "dst", Type: "string"}}},
			"move":      {Name: "move", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "src", Type: "string"}, {Name: "dst", Type: "string"}}},
			"Copy":      {Name: "Copy", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "src", Type: "string"}, {Name: "dst", Type: "string"}}},
			"copy":      {Name: "copy", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "src", Type: "string"}, {Name: "dst", Type: "string"}}},
			"Write":     {Name: "Write", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}, {Name: "content", Type: "string"}}},
			"write":     {Name: "write", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "path", Type: "string"}, {Name: "content", Type: "string"}}},
			"EnsureDir": {Name: "EnsureDir", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"ensureDir": {Name: "ensureDir", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"ensuredir": {Name: "ensuredir", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
		},
	}
	stdUnit := &Symbol{
		Name: "std",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"log":     {Name: "log", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "any"}}, IsVariadic: true, MinArgs: 1},
			"Log":     {Name: "Log", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "any"}}, IsVariadic: true, MinArgs: 1},
			"println": {Name: "println", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "any"}}, IsVariadic: true, MinArgs: 1},
			"Println": {Name: "Println", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "any"}}, IsVariadic: true, MinArgs: 1},
			"print":   {Name: "print", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "any"}}, IsVariadic: true, MinArgs: 1},
			"Print":   {Name: "Print", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "any"}}, IsVariadic: true, MinArgs: 1},
			"info":    {Name: "info", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"Info":    {Name: "Info", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"success": {Name: "success", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"Success": {Name: "Success", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"warn":    {Name: "warn", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"Warn":    {Name: "Warn", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"error":   {Name: "error", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"Error":   {Name: "Error", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "message", Type: "string"}}},
			"sleep":   {Name: "sleep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "duration", Type: "any"}}},
			"Sleep":   {Name: "Sleep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "duration", Type: "any"}}},
			"fatal":   {Name: "fatal", Kind: SymFunc, Type: "never", Params: []*ast.Parameter{{Name: "message", Type: "any"}, {Name: "code", Type: "number"}}, MinArgs: 1, HasOptional: true},
			"Fatal":   {Name: "Fatal", Kind: SymFunc, Type: "never", Params: []*ast.Parameter{{Name: "message", Type: "any"}, {Name: "code", Type: "number"}}, MinArgs: 1, HasOptional: true},
			"type":    {Name: "type", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "value", Type: "any"}}},
			"Type":    {Name: "Type", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "value", Type: "any"}}},
			"typeof":  {Name: "typeof", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "value", Type: "any"}}},
			"Typeof":  {Name: "Typeof", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "value", Type: "any"}}},
		},
	}
	sysUnit := &Symbol{
		Name: "Sys",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"GetEnv":  {Name: "GetEnv", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "key", Type: "string"}}},
			"getEnv":  {Name: "getEnv", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "key", Type: "string"}}},
			"Env":     {Name: "Env", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "key", Type: "string"}}},
			"env":     {Name: "env", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "key", Type: "string"}}},
			"Time":    {Name: "Time", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"time":    {Name: "time", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"Os":      {Name: "Os", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"os":      {Name: "os", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"Arch":    {Name: "Arch", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"arch":    {Name: "arch", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"Sleep":   {Name: "Sleep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "duration", Type: "any"}}},
			"sleep":   {Name: "sleep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "duration", Type: "any"}}},
			"Exit":    {Name: "Exit", Kind: SymFunc, Type: "never", Params: []*ast.Parameter{{Name: "code", Type: "number"}}, MinArgs: 0, HasOptional: true},
			"exit":    {Name: "exit", Kind: SymFunc, Type: "never", Params: []*ast.Parameter{{Name: "code", Type: "number"}}, MinArgs: 0, HasOptional: true},
			"require": {Name: "require", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "constraint", Type: "string"}, {Name: "options", Type: "ROpt", Optional: true}}, MinArgs: 1, HasOptional: true},
			"Require": {Name: "Require", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "constraint", Type: "string"}, {Name: "options", Type: "ROpt", Optional: true}}, MinArgs: 1, HasOptional: true},
			"version": {Name: "version", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"Version": {Name: "Version", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"cwd":     {Name: "cwd", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"Cwd":     {Name: "Cwd", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"homeDir": {Name: "homeDir", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"args":    {Name: "args", Kind: SymFunc, Type: "list", Params: []*ast.Parameter{}},
			"Args":    {Name: "Args", Kind: SymFunc, Type: "list", Params: []*ast.Parameter{}},
			"hasArg":  {Name: "hasArg", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "flags", Type: "string"}}, MinArgs: 0, IsVariadic: true},
			"HasArg":  {Name: "HasArg", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "flags", Type: "string"}}, MinArgs: 0, IsVariadic: true},
		},
	}
	cmdUnit := &Symbol{
		Name: "Cmd",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Exec": {Name: "Exec", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "command", Type: "string"}, {Name: "dir", Type: "string"}}, MinArgs: 1, HasOptional: true},
			"exec": {Name: "exec", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "command", Type: "string"}, {Name: "dir", Type: "string"}}, MinArgs: 1, HasOptional: true},
		},
	}
	cryptoUnit := &Symbol{
		Name: "Crypto",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"GetRandomBytes": {Name: "GetRandomBytes", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"getRandomBytes": {Name: "getRandomBytes", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"RandomBytes":    {Name: "RandomBytes", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"randomBytes":    {Name: "randomBytes", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"RandomHex":      {Name: "RandomHex", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"randomHex":      {Name: "randomHex", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"RandomBase64":   {Name: "RandomBase64", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"randomBase64":   {Name: "randomBase64", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "length", Type: "number"}}},
			"Sha256":         {Name: "Sha256", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "data", Type: "string"}}},
			"sha256":         {Name: "sha256", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "data", Type: "string"}}},
			"Uuid":           {Name: "Uuid", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
			"uuid":           {Name: "uuid", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{}},
		},
	}

	localUnit := &Symbol{
		Name: "Local",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Keep":       {Name: "Keep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "files", Type: "list"}}},
			"Drop":       {Name: "Drop", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "patterns", Type: "list"}}},
			"UseVariant": {Name: "UseVariant", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "variant", Type: "string"}}},
		},
	}

	pathUnit := &Symbol{
		Name: "Path",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Join": {Name: "Join", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}, IsVariadic: true, MinArgs: 1},
			"join": {Name: "join", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}, IsVariadic: true, MinArgs: 1},
			"Dir":  {Name: "Dir", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"dir":  {Name: "dir", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"Base": {Name: "Base", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"base": {Name: "base", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"Ext":  {Name: "Ext", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
			"ext":  {Name: "ext", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "path", Type: "string"}}},
		},
	}
	jsonUnit := &Symbol{
		Name: "Json",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Parse":     {Name: "Parse", Kind: SymFunc, Type: "any", Params: []*ast.Parameter{{Name: "jsonStr", Type: "string"}}},
			"parse":     {Name: "parse", Kind: SymFunc, Type: "any", Params: []*ast.Parameter{{Name: "jsonStr", Type: "string"}}},
			"Stringify": {Name: "Stringify", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "val", Type: "any"}}},
			"stringify": {Name: "stringify", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "val", Type: "any"}}},
		},
	}
	mathUnit := &Symbol{
		Name: "Math",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Round":  {Name: "Round", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"round":  {Name: "round", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"Floor":  {Name: "Floor", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"floor":  {Name: "floor", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"Ceil":   {Name: "Ceil", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"ceil":   {Name: "ceil", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"Abs":    {Name: "Abs", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"abs":    {Name: "abs", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "n", Type: "number"}}},
			"Min":    {Name: "Min", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "a", Type: "number"}, {Name: "b", Type: "number"}}},
			"min":    {Name: "min", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "a", Type: "number"}, {Name: "b", Type: "number"}}},
			"Max":    {Name: "Max", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "a", Type: "number"}, {Name: "b", Type: "number"}}},
			"max":    {Name: "max", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "a", Type: "number"}, {Name: "b", Type: "number"}}},
			"Random": {Name: "Random", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{}},
			"random": {Name: "random", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{}},
		},
	}

	c.availableModules["Net"] = netUnit
	c.availableModules["Fs"] = fsUnit
	c.availableModules["std"] = stdUnit
	c.availableModules["Std"] = stdUnit
	c.availableModules["Sys"] = sysUnit
	c.availableModules["sys"] = sysUnit
	c.availableModules["Cmd"] = cmdUnit
	c.availableModules["cmd"] = cmdUnit
	c.availableModules["Crypto"] = cryptoUnit
	c.availableModules["Local"] = localUnit
	c.availableModules["Path"] = pathUnit
	c.availableModules["path"] = pathUnit
	c.availableModules["Json"] = jsonUnit
	c.availableModules["json"] = jsonUnit
	c.availableModules["Math"] = mathUnit
	c.availableModules["math"] = mathUnit

	sigUnit := &Symbol{
		Name: "sig",
		Kind: SymUnit,
		Properties: map[string]*Symbol{
			"interrupt": {Name: "interrupt", Kind: SymVar, Type: "string"},
			"terminate": {Name: "terminate", Kind: SymVar, Type: "string"},
			"hangup":    {Name: "hangup", Kind: SymVar, Type: "string"},
			"exit":      {Name: "exit", Kind: SymVar, Type: "string"},
			"error":     {Name: "error", Kind: SymVar, Type: "string"},
		},
		Methods: make(map[string]*Symbol),
	}
	c.availableModules["sig"] = sigUnit
	c.availableModules["Sig"] = sigUnit

	stdInterfaces := []string{
		"Document", "SysNative", "FsNative", "NetNative", "CryptoNative",
		"JsonNative", "MathNative", "DateNative", "CmdNative", "PathNative", "StdNative",
	}
	for _, ifaceName := range stdInterfaces {
		c.interfaces[ifaceName] = &ast.InterfaceStatement{Name: ifaceName}
	}

	c.interfaces["CmdResult"] = &ast.InterfaceStatement{
		Name: "CmdResult",
		Fields: []*ast.InterfaceField{
			{Name: "out", Type: "string"},
			{Name: "err", Type: "string"},
			{Name: "code", Type: "number"},
			{Name: "ok", Type: "bool"},
		},
	}

	c.interfaces["ROpt"] = &ast.InterfaceStatement{
		Name: "ROpt",
		Fields: []*ast.InterfaceField{
			{Name: "onMismatch", Type: "\"fatal\" or \"warn\" or \"auto-update\"", Optional: true},
			{Name: "mode", Type: "\"log\" or \"silent\"", Optional: true},
		},
	}

	dateMapType := "@map{ iso: string, timestamp: number, year: number, month: number, day: number, hour: number, minute: number, second: number, weekday: string }"
	dateUnit := &Symbol{
		Name: "date",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"now":       {Name: "now", Kind: SymFunc, Type: dateMapType},
			"Now":       {Name: "Now", Kind: SymFunc, Type: dateMapType},
			"iso":       {Name: "iso", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"Iso":       {Name: "Iso", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"today":     {Name: "today", Kind: SymFunc, Type: "string"},
			"Today":     {Name: "Today", Kind: SymFunc, Type: "string"},
			"timestamp": {Name: "timestamp", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"Timestamp": {Name: "Timestamp", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"format":    {Name: "format", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "dateOrFormat", Type: "any"}, {Name: "format", Type: "string", Optional: true}}, MinArgs: 1},
			"Format":    {Name: "Format", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "dateOrFormat", Type: "any"}, {Name: "format", Type: "string", Optional: true}}, MinArgs: 1},
			"parse":     {Name: "parse", Kind: SymFunc, Type: dateMapType, Params: []*ast.Parameter{{Name: "dateStr", Type: "string"}, {Name: "format", Type: "string", Optional: true}}, MinArgs: 1},
			"Parse":     {Name: "Parse", Kind: SymFunc, Type: dateMapType, Params: []*ast.Parameter{{Name: "dateStr", Type: "string"}, {Name: "format", Type: "string", Optional: true}}, MinArgs: 1},
			"add":       {Name: "add", Kind: SymFunc, Type: dateMapType, Params: []*ast.Parameter{{Name: "date", Type: "any"}, {Name: "amount", Type: "any"}, {Name: "unit", Type: "string", Optional: true}}, MinArgs: 2},
			"Add":       {Name: "Add", Kind: SymFunc, Type: dateMapType, Params: []*ast.Parameter{{Name: "date", Type: "any"}, {Name: "amount", Type: "any"}, {Name: "unit", Type: "string", Optional: true}}, MinArgs: 2},
			"diff":      {Name: "diff", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "dateA", Type: "any"}, {Name: "dateB", Type: "any"}, {Name: "unit", Type: "string", Optional: true}}, MinArgs: 2},
			"Diff":      {Name: "Diff", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "dateA", Type: "any"}, {Name: "dateB", Type: "any"}, {Name: "unit", Type: "string", Optional: true}}, MinArgs: 2},
			"year":      {Name: "year", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"Year":      {Name: "Year", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"month":     {Name: "month", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"Month":     {Name: "Month", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"day":       {Name: "day", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
			"Day":       {Name: "Day", Kind: SymFunc, Type: "number", Params: []*ast.Parameter{{Name: "date", Type: "any", Optional: true}}},
		},
	}
	c.availableModules["Date"] = dateUnit
	c.availableModules["date"] = dateUnit

	topLevelFuncs := map[string]*Symbol{
		"Drop":        {Name: "Drop", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "patterns", Type: "list"}}},
		"drop":        {Name: "drop", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "patterns", Type: "list"}}},
		"Keep":        {Name: "Keep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "files", Type: "list"}}},
		"keep":        {Name: "keep", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "files", Type: "list"}}},
		"replace":     {Name: "replace", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "target", Type: "string"}, {Name: "replacement", Type: "string"}}},
		"replaceLine": {Name: "replaceLine", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "pattern", Type: "string"}, {Name: "replacement", Type: "string"}}},
		"remove":      {Name: "remove", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "target", Type: "string"}}},
		"removeLines": {Name: "removeLines", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "pattern", Type: "string"}}},
		"removeBlock": {Name: "removeBlock", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "start", Type: "string"}, {Name: "end", Type: "string"}}},
		"removeUntil": {Name: "removeUntil", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "start", Type: "string"}, {Name: "until", Type: "string"}}},
		"commentOut":  {Name: "commentOut", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "pattern", Type: "string"}}},
		"append":      {Name: "append", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "content", Type: "string"}}},
		"prepend":     {Name: "prepend", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "content", Type: "string"}}},
		"jsonSet":     {Name: "jsonSet", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "keyPath", Type: "string"}, {Name: "value", Type: "any"}}},
		"jsonRemove":  {Name: "jsonRemove", Kind: SymFunc, Type: "void", Params: []*ast.Parameter{{Name: "keyPath", Type: "string"}}},
	}
	for _, sym := range topLevelFuncs {
		c.scope.Define(sym)
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

func CreateDocumentSymbol(name string) *Symbol {
	return &Symbol{
		Name: name,
		Kind: SymVar,
		Type: "Document",
		Methods: map[string]*Symbol{
			"getContent":     {Name: "getContent", Kind: SymFunc, Type: "string"},
			"setContent":     {Name: "setContent", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "newContent", Type: "string"}}},
			"append":         {Name: "append", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "content", Type: "string"}}},
			"prepend":        {Name: "prepend", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "content", Type: "string"}}},
			"contains":       {Name: "contains", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "pattern", Type: "string"}}},
			"size":           {Name: "size", Kind: SymFunc, Type: "number"},
			"getLines":       {Name: "getLines", Kind: SymFunc, Type: "list[string]"},
			"setLines":       {Name: "setLines", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "lines", Type: "list"}}},
			"getLine":        {Name: "getLine", Kind: SymFunc, Type: "string", Params: []*ast.Parameter{{Name: "lineNum", Type: "number"}}},
			"setLine":        {Name: "setLine", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "lineNum", Type: "number"}, {Name: "text", Type: "string"}}},
			"removeLine":     {Name: "removeLine", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "lineNumOrPattern", Type: "any"}}, MinArgs: 1},
			"removeLines":    {Name: "removeLines", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "patternOrStart", Type: "any"}}, MinArgs: 1, IsVariadic: true},
			"getJson":        {Name: "getJson", Kind: SymFunc, Type: "any", Params: []*ast.Parameter{{Name: "keyPath", Type: "string"}}},
			"setJson":        {Name: "setJson", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "keyPath", Type: "string"}, {Name: "value", Type: "any"}}},
			"removeJson":     {Name: "removeJson", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "keyPath", Type: "string"}}},
			"replace":        {Name: "replace", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "target", Type: "string"}, {Name: "replacement", Type: "string"}}},
			"replaceFuzzy":   {Name: "replaceFuzzy", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "target", Type: "string"}, {Name: "replacement", Type: "string"}}},
			"remove":         {Name: "remove", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "target", Type: "string"}}},
			"removeFuzzy":    {Name: "removeFuzzy", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "target", Type: "string"}}},
			"insertAfter":    {Name: "insertAfter", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "anchor", Type: "string"}, {Name: "content", Type: "string"}}},
			"insertBefore":   {Name: "insertBefore", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "anchor", Type: "string"}, {Name: "content", Type: "string"}}},
			"removeBetween":  {Name: "removeBetween", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "startPattern", Type: "string"}, {Name: "endPattern", Type: "string"}}, MinArgs: 2, IsVariadic: true},
			"removeBalanced": {Name: "removeBalanced", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{{Name: "startAnchor", Type: "string"}}, MinArgs: 1, IsVariadic: true},
			"cleanGaps":      {Name: "cleanGaps", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{}, MinArgs: 0},
			"cleanGap":       {Name: "cleanGap", Kind: SymFunc, Type: "bool", Params: []*ast.Parameter{}, MinArgs: 0},
			"save":           {Name: "save", Kind: SymFunc, Type: "bool"},
		},
	}
}

type NativeParam struct {
	Name     string
	Type     string
	Optional bool
}

type NativeMethodSpec struct {
	Params     []NativeParam
	ReturnType string
}

var StringMethodSpecs = map[string]NativeMethodSpec{
	"type": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"typeof": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"len": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"length": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"trim": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"trimleft": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"trimright": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"toupper": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"uppercase": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"tolower": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"lowercase": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"contains": {
		Params: []NativeParam{
			{Name: "sub", Type: "string"},
		},
		ReturnType: "bool",
	},
	"startswith": {
		Params: []NativeParam{
			{Name: "pfx", Type: "string"},
		},
		ReturnType: "bool",
	},
	"endswith": {
		Params: []NativeParam{
			{Name: "sfx", Type: "string"},
		},
		ReturnType: "bool",
	},
	"split": {
		Params: []NativeParam{
			{Name: "sep", Type: "string"},
		},
		ReturnType: "list[string]",
	},
	"replace": {
		Params: []NativeParam{
			{Name: "old", Type: "string"},
			{Name: "new", Type: "string"},
		},
		ReturnType: "string",
	},
	"slice": {
		Params: []NativeParam{
			{Name: "start", Type: "number"},
			{Name: "end", Type: "number", Optional: true},
		},
		ReturnType: "string",
	},
	"repeat": {
		Params: []NativeParam{
			{Name: "count", Type: "number"},
		},
		ReturnType: "string",
	},
	"tostring": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"to_string": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
}

var ListMethodSpecs = map[string]NativeMethodSpec{
	"type": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"typeof": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"len": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"length": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"join": {
		Params: []NativeParam{
			{Name: "sep", Type: "string", Optional: true},
		},
		ReturnType: "string",
	},
	"contains": {
		Params: []NativeParam{
			{Name: "item", Type: "any"},
		},
		ReturnType: "bool",
	},
	"push": {
		Params: []NativeParam{
			{Name: "item", Type: "any"},
		},
		ReturnType: "void",
	},
	"pop": {
		Params:     []NativeParam{},
		ReturnType: "any",
	},
	"reverse": {
		Params:     []NativeParam{},
		ReturnType: "list",
	},
	"slice": {
		Params: []NativeParam{
			{Name: "start", Type: "number"},
			{Name: "end", Type: "number", Optional: true},
		},
		ReturnType: "list",
	},
	"get": {
		Params: []NativeParam{
			{Name: "index", Type: "number"},
		},
		ReturnType: "any",
	},
	"at": {
		Params: []NativeParam{
			{Name: "index", Type: "number"},
		},
		ReturnType: "any",
	},
}

func IsStringMethod(name string) bool {
	_, ok := StringMethodSpecs[strings.ToLower(name)]
	return ok
}

func IsListMethod(name string) bool {
	_, ok := ListMethodSpecs[strings.ToLower(name)]
	return ok
}

func GetStringMethodSpec(name string) (NativeMethodSpec, bool) {
	spec, ok := StringMethodSpecs[strings.ToLower(name)]
	return spec, ok
}

func GetListMethodSpec(name string) (NativeMethodSpec, bool) {
	spec, ok := ListMethodSpecs[strings.ToLower(name)]
	return spec, ok
}

var NumberMethodSpecs = map[string]NativeMethodSpec{
	"type": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"typeof": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"abs": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"floor": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"ceil": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"round": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"int": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"tostring": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"to_string": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"fixed": {
		Params: []NativeParam{
			{Name: "digits", Type: "number", Optional: true},
		},
		ReturnType: "string",
	},
}

func IsNumberMethod(name string) bool {
	_, ok := NumberMethodSpecs[strings.ToLower(name)]
	return ok
}

func GetNumberMethodSpec(name string) (NativeMethodSpec, bool) {
	spec, ok := NumberMethodSpecs[strings.ToLower(name)]
	return spec, ok
}

var MapMethodSpecs = map[string]NativeMethodSpec{
	"type": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"typeof": {
		Params:     []NativeParam{},
		ReturnType: "string",
	},
	"get": {
		Params: []NativeParam{
			{Name: "key", Type: "string"},
			{Name: "default", Type: "any", Optional: true},
		},
		ReturnType: "any",
	},
	"set": {
		Params: []NativeParam{
			{Name: "key", Type: "string"},
			{Name: "value", Type: "any"},
		},
		ReturnType: "map",
	},
	"merge": {
		Params: []NativeParam{
			{Name: "other", Type: "map"},
		},
		ReturnType: "map",
	},
	"keys": {
		Params:     []NativeParam{},
		ReturnType: "list[string]",
	},
	"values": {
		Params:     []NativeParam{},
		ReturnType: "list",
	},
	"entries": {
		Params:     []NativeParam{},
		ReturnType: "list",
	},
	"has": {
		Params: []NativeParam{
			{Name: "key", Type: "string"},
		},
		ReturnType: "bool",
	},
	"len": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"size": {
		Params:     []NativeParam{},
		ReturnType: "number",
	},
	"delete": {
		Params: []NativeParam{
			{Name: "key", Type: "string"},
		},
		ReturnType: "void",
	},
	"clear": {
		Params:     []NativeParam{},
		ReturnType: "void",
	},
}

func IsMapMethod(name string) bool {
	_, ok := MapMethodSpecs[strings.ToLower(name)]
	return ok
}

func GetMapMethodSpec(name string) (NativeMethodSpec, bool) {
	spec, ok := MapMethodSpecs[strings.ToLower(name)]
	return spec, ok
}
