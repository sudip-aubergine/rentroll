package rlib

import (
	"extres"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/kardianos/osext"
)

// AppConfig is the shared struct of configuration values
var AppConfig extres.ExternalResources

// RRReadConfig will read the configuration file "config.json" if
// it exists in the current directory
func RRReadConfig(fPath ...string) error {
	var (
		folderPath string
		err        error
		expath     string
		adjustEnv  bool
	)
	adjustEnv = false
	expath, err = osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}
	// as of now, just limit the parameters upto 1 length only
	Console("A\n")
	if len(fPath) > 0 && len(fPath[0]) > 0 {
		Console("B len(fPath) = %d, fPath = %s\n", len(fPath), fPath)
		folderPath = fPath[0]
		adjustEnv = folderPath == expath // is it in the release directory
	} else {
		Console("C\n")
		folderPath = expath
		if len(folderPath) == 0 {
			Console("D\n")
			folderPath = "."
		}
		Console("E\n")
		adjustEnv = true
	}

	fname := path.Join(folderPath, "config.json")
	Console("ReadConfig( %q ),  expath = %s, folderpath = %s\n", fname, expath, folderPath)
	err = extres.ReadConfig(fname, &AppConfig)
	if err != nil {
		log.Fatal(err)
	}

	//----------------------------------------------------------------------
	// This ensures that config.json in the server's directory is the only
	// one that can set the Environment to be extres.APPENVPROD
	//----------------------------------------------------------------------
	if !adjustEnv {
		AppConfig.Env = extres.APPENVDEV
	}

	RRdb.Zone, err = time.LoadLocation(AppConfig.Timezone)
	if err != nil {
		fmt.Printf("Error loading timezone %s : %s\n", AppConfig.Timezone, err.Error())
		Ulog("Error loading timezone %s : %s", AppConfig.Timezone, err.Error())
	}
	return err
}
