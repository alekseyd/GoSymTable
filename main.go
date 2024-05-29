package main

import (
	"context"
	"debug/buildinfo"
	"debug/elf"
	"flag"
	"fmt"
	stdLog "log"
	"os"
	"runtime/debug"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"gitlab.com/Raven-IO/GoSymTable/symbol/addr2line"
	"gitlab.com/Raven-IO/GoSymTable/symbol/elfutils"
)

const callStackDepth = 5

func main() {
	_ = context.Background()

	lvl := level.AllowDebug()
	writer := log.NewSyncWriter(os.Stdout)
	logger := log.NewLogfmtLogger(writer)
	logger = level.NewFilter(logger, lvl)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.Caller(callStackDepth))

	flag.Parse()
	if len(flag.Args()) <= 0 {
		stdLog.Fatalf("no elf file provided")
	}
	file := flag.Args()[0]
	stdLog.Println(file)

	e, err := elf.Open(file)
	if err != nil {
		level.Error(logger).Log("msg", "can't open elf", "file", file, "err", err)
		os.Exit(1)
	}

	defer e.Close()

	if elfutils.HasGoPclntab(e) {
		lnr, err := addr2line.Go(logger, file, e)
		if err != nil {
			level.Error(logger).Log("msg", "can't create liner", "file", file, "err", err)
		}

		level.Info(logger).Log("len(Syms)", len(lnr.Symtab.Syms))
		level.Info(logger).Log("len(Funcs)", len(lnr.Symtab.Funcs))
		level.Info(logger).Log("len(Objs)", len(lnr.Symtab.Objs))
		level.Info(logger).Log("len(Files)", len(lnr.Symtab.Files))

		level.Info(logger).Log("msg", "File Names")
		for f := range lnr.Symtab.Files {
			fmt.Println(f)
		}

		fmt.Println("")

		rtPackages := make(map[string]bool)
		//level.Info(logger).Log("msg", "Funcs")
		for _, f := range lnr.Symtab.Funcs {
			//fmt.Printf("func=%d, name=%s, package=%s, receiver=%s", i, f.BaseName(), f.PackageName(), f.ReceiverName())
			//fmt.Println("")
			rtPackages[f.PackageName()] = true
		}

		level.Info(logger).Log("msg", "In-line packages")
		for name, _ := range rtPackages {
			fmt.Printf("package=%s", name)
			fmt.Println("")
		}
	}

	bi, err := buildinfo.ReadFile(file)
	if err != nil {
		level.Error(logger).Log("msg", "can't open buildinfo", "file", file, "err", err)
		os.Exit(1)
	}

	//	level.Info(logger).Log("buildinfo", bi.String())

	mods := make(map[string]debug.Module)
	if bi.Main != (debug.Module{}) {
		if bi.Main.Replace != nil {
			mods[bi.Main.Path] = *bi.Main.Replace
		} else {
			mods[bi.Main.Path] = bi.Main
		}
	}

	level.Info(logger).Log("buildinfo", "q1")
	for _, m := range bi.Deps {
		if *m != (debug.Module{}) {
			mods[m.Path] = *m
		}
	}

	level.Info(logger).Log("buildinfo", "q1")
	level.Info(logger).Log("msg", "Modules")
	for name, m := range mods {
		fmt.Printf("module=%v, version=%s, hash=%s", name, m.Version, m.Sum)
		fmt.Println("")
	}

}
