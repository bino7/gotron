package main

import (
	"fmt"
	"github.com/Benchkram/errz"
	"github.com/bino7/gotron/cmd/gotron-builder/internal/application"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var gotronBuilderVersion = "undefined"

func init() {
	// General
	rootCmd.PersistentFlags().StringP("name", "n", "gotron-app",
		"App name")
	rootCmd.PersistentFlags().StringP("go", "g", ".",
		"Go entrypoint, must point to a directory containing a main.go")
	rootCmd.PersistentFlags().StringP("app", "a", ".gotron/assets/",
		"Application directory, must point to a directory containing a webapp starting at index.html")
	rootCmd.PersistentFlags().StringP("out", "", ".",
		"Application output directory. Build output will be put in dist/* inside this directory.")
	rootCmd.PersistentFlags().BoolP("version", "v", false,
		"Returns gotron-builder version")
	rootCmd.PersistentFlags().Bool("polymer", false,
		"run polymer build to the src")
	rootCmd.PersistentFlags().Bool("no-prune", false,
		"add --no-prune flag when running electron-package")

	// Electron-Builder parameters

	// Platforms

	// Build for macOS
	rootCmd.PersistentFlags().BoolP("mac", "m", false, "Build for macOS")
	rootCmd.PersistentFlags().BoolP("macos", "o", false, "Build for macOS")

	// Build for linux
	rootCmd.PersistentFlags().BoolP("linux", "l", false, "Build for Linux")

	// Build for windows
	rootCmd.PersistentFlags().BoolP("win", "w", false, "Build for Windows")
	rootCmd.PersistentFlags().BoolP("windows", "", false, "Build for Windows")

	// Architectures
	rootCmd.PersistentFlags().BoolP("x64", "", false, "Build for x64")
	rootCmd.PersistentFlags().BoolP("ia32", "", false, "Build for ia32")
	rootCmd.PersistentFlags().BoolP("armv7l", "", false, "Build for armv7l")
	rootCmd.PersistentFlags().BoolP("arm64", "", false, "Build for arm64")
}

func Run(cmd *cobra.Command, args []string) {
	zerolog.TimeFieldFormat = ""
	log.Logger.Level(zerolog.ErrorLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Caller().Logger()

	app, err := parseFlags(cmd)
	if err != nil {
		log.Fatal().Msg("Bad input parameters:-")
		log.Fatal().Msg(err.Error())
		return
	}

	if err := app.Run(); err != nil {
		panic(err)
		log.Fatal().Msg(err.Error())
	}
}

var rootCmd = &cobra.Command{
	Use:   "gotron-builder",
	Short: "build executables using gotron",
	Long:  ``,
	Run:   Run,
}

func parseFlags(cmd *cobra.Command) (app *application.App, err error) {
	defer errz.Recover(&err)

	//General
	name := cmd.Flag("name").Value.String()
	goDir := cmd.Flag("go").Value.String()
	appDir := cmd.Flag("app").Value.String()
	outputDir := cmd.Flag("out").Value.String()
	version, _ := strconv.ParseBool(cmd.Flag("version").Value.String())

	// make paths absolute
	appDir, err = filepath.Abs(appDir)
	errz.Fatal(err)
	goDir, err = filepath.Abs(goDir)
	errz.Fatal(err)
	outputDir, err = filepath.Abs(outputDir)
	errz.Fatal(err)

	polymer, _ := strconv.ParseBool(cmd.Flag("polymer").Value.String())
	noPrune, _ := strconv.ParseBool(cmd.Flag("no-prune").Value.String())

	//Target Platform
	m1, _ := strconv.ParseBool(cmd.Flag("mac").Value.String())
	m2, _ := strconv.ParseBool(cmd.Flag("macos").Value.String())
	mac := m1 || m2

	linux, _ := strconv.ParseBool(cmd.Flag("linux").Value.String())

	w1, _ := strconv.ParseBool(cmd.Flag("windows").Value.String())
	w2, _ := strconv.ParseBool(cmd.Flag("win").Value.String())
	windows := w1 || w2

	// Architectures
	arch := make(map[string]bool)
	arch["x64"], _ = strconv.ParseBool(cmd.Flag("x64").Value.String())       //GOARCH=amd64
	arch["ia32"], _ = strconv.ParseBool(cmd.Flag("ia32").Value.String())     //GOARCH=386
	arch["armv7l"], _ = strconv.ParseBool(cmd.Flag("armv7l").Value.String()) //GOARCH=arm GOARM=7
	arch["arm64"], _ = strconv.ParseBool(cmd.Flag("arm64").Value.String())   //GOARCH=arm64

	// Go build

	// If version is set just print it and exit.
	if version {
		fmt.Printf("goton-builder %s \n", gotronBuilderVersion)
		os.Exit(0)
	}

	// Create App and set values
	app = application.New()

	// TODO allow selecting multiple values for arch and platform
	if (windows && linux) || (windows && mac) || (mac && linux) {
		log.Error().Msg("Only one target platform is allowed at a time.")
		return
	}

	if windows {
		err = app.SetTarget("win")
	} else if linux {
		err = app.SetTarget("linux")
	} else if mac {
		err = app.SetTarget("mac")
	}
	errz.Log(err)

	archCount := 0
	app.Arch = "x64" //default value
	for k, v := range arch {
		if v {
			app.Arch = k
			archCount++
		}
	}

	if archCount > 1 {
		log.Error().Msg("Only one target architecture is allowed at a time.")
		return
	}

	app.Name = name
	app.GoEntryPoint = goDir
	app.AppDir = appDir
	app.OutputDir = outputDir
	app.Polymer = polymer
	app.NoPrune = noPrune

	return
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}
	/*app := application.New()
	app.Name="dashboard"
	app.GoEntryPoint="/Users/bino/go/src/github.com/bino7/dashboard"
	app.AppDir="/Users/bino/go/src/github.com/bino7/dashboard/ui"
	app.OutputDir="/Users/bino/go/src/github.com/bino7/dashboard/dist"
	//app.Polymer=true
	app.SetTarget("mac")
	app.Arch = "x64"
	app.Run()*/
}
