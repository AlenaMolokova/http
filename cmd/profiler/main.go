package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/sirupsen/logrus"
)

func benchmarkLoad(appInstance *app.App, count int) {
	appInstance.GenerateTestLoad(count)
}

func main() {
	var memProfileName string
	var cpuProfileName string
	var testLoad int
	var profileMode string
	var allocsProfileName string

	flag.StringVar(&allocsProfileName, "allocsprofile", "", "write allocs profile to file (optional)")
	flag.StringVar(&memProfileName, "profile", "base.pprof", "Heap profile file name (default: base.pprof)")
	flag.StringVar(&cpuProfileName, "cpuprofile", "", "CPU profile file name (optional)")
	flag.IntVar(&testLoad, "load", 1000, "Number of URLs to generate for testing")
	flag.StringVar(&profileMode, "mode", "base", "Profile mode: 'base' or 'result'")
	flag.Parse()

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	if err := os.MkdirAll("profiles", os.ModePerm); err != nil {
		logrus.WithError(err).Fatal("Failed to create profiles directory")
	}

	if profileMode == "result" {
		memProfileName = "result.pprof"
	} else {
		memProfileName = "base.pprof"
	}

	memProfilePath := filepath.Join("profiles", memProfileName)

	logrus.WithFields(logrus.Fields{
		"mode":      profileMode,
		"profile":   memProfileName,
		"test_load": testLoad,
		"heap_path": memProfilePath,
		"cpu_path":  cpuProfileName,
	}).Info("Starting profiling")

	var cpuProfileFile *os.File
	if cpuProfileName != "" {
		cpuPath := filepath.Join("profiles", cpuProfileName)
		f, err := os.Create(cpuPath)
		if err != nil {
			logrus.WithError(err).Fatal("Could not create CPU profile")
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			logrus.WithError(err).Fatal("Could not start CPU profile")
		}
		logrus.WithField("file", cpuPath).Info("CPU profiling started")
		cpuProfileFile = f
	}

	cfg := config.NewConfig()
	logrus.WithField("config", cfg).Info("Configuration loaded")

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize application")
	}
	logrus.Info("Application initialized")

	if testLoad > 0 {
		logrus.Infof("Generating test load: %d URLs", testLoad)
		benchmarkLoad(appInstance, testLoad)
	}

	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		cpuProfileFile.Close()
		logrus.WithField("file", cpuProfileName).Info("CPU profiling stopped")
	}

	f, err := os.Create(memProfilePath)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create memory profile")
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		logrus.WithError(err).Fatal("Could not write memory profile")
	}
	logrus.Infof("Heap profile written to %s", memProfilePath)

	if allocsProfileName != "" {
		allocsPath := filepath.Join("profiles", allocsProfileName)
		f, err := os.Create(allocsPath)
		if err != nil {
			logrus.WithError(err).Fatal("Could not create allocs profile")
		}
		defer f.Close()

		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
			logrus.WithError(err).Fatal("Could not write allocs profile")
		}
		logrus.Infof("Allocs profile written to %s", allocsPath)
	}

	if profileMode == "base" {
		logrus.Info("==========================================")
		logrus.Info("STEP 1 COMPLETE: Base profile has been created")
		logrus.Info("To analyze: go tool pprof -http=:8080 profiles/base.pprof")
		logrus.Info("Then run: go run cmd/profiler/main.go -mode=result")
		logrus.Info("==========================================")
	} else {
		logrus.Info("==========================================")
		logrus.Info("STEP 3 COMPLETE: Result profile created")
		logrus.Info("Compare profiles: go tool pprof -http=:8080 -diff_base=profiles/base.pprof profiles/result.pprof")
		logrus.Info("==========================================")
	}
}
