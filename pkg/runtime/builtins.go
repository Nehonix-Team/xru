package runtime

import (
	"sync"
)

var (
	globalCliArgsMu sync.RWMutex
	globalCliArgs   []string
)

func SetGlobalCliArgs(args []string) {
	globalCliArgsMu.Lock()
	defer globalCliArgsMu.Unlock()
	globalCliArgs = args
}

func GetGlobalCliArgs() []string {
	globalCliArgsMu.RLock()
	defer globalCliArgsMu.RUnlock()
	return globalCliArgs
}

func NewBuiltinUnits() map[string]*UnitInstance {
	units := make(map[string]*UnitInstance)

	// 1. #UNIT std (Standard Library I/O & Output)
	stdUnit := newStdUnit()
	units["std"] = stdUnit
	units["Std"] = stdUnit

	// 2. #UNIT Fs
	fsUnit := newFsUnit()
	units["Fs"] = fsUnit
	units["fs"] = fsUnit

	// 3. #UNIT Net
	netUnit := newNetUnit()
	units["Net"] = netUnit
	units["net"] = netUnit

	// 4. #UNIT Sys
	sysUnit := newSysUnit()
	units["Sys"] = sysUnit
	units["sys"] = sysUnit

	// 5. #UNIT Cmd (exécution de commandes terminal cross-platform)
	cmdUnit := newCmdUnit()
	units["Cmd"] = cmdUnit
	units["cmd"] = cmdUnit

	// 6. #UNIT Crypto (génération cryptographique sécurisée)
	cryptoUnit := newCryptoUnit()
	units["Crypto"] = cryptoUnit

	// 7. #UNIT Json
	jsonUnit := newJsonUnit()
	units["Json"] = jsonUnit

	// 8. #UNIT Math
	mathUnit := newMathUnit()
	units["Math"] = mathUnit

	// 9. #UNIT Path
	pathUnit := newPathUnit()
	units["Path"] = pathUnit
	units["path"] = pathUnit

	// 10. #UNIT date (Date & Time manipulation utilities)
	dateUnit := newDateUnit()
	units["Date"] = dateUnit
	units["date"] = dateUnit

	// 11. #UNIT sig (Signal handling & process lifecycle)
	sigUnit := newSigUnit()
	units["Sig"] = sigUnit
	units["sig"] = sigUnit

	return units
}
