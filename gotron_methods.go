// +build !gotronpack

package gotron

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/bino7/gotron/internal/file"
	"github.com/pkg/errors"

	"github.com/otiai10/copy"

	"github.com/Benchkram/errz"
)

const (
	templateApplicationDir = "templates/app"
	gotronFileMode         = os.FileMode(0755)
)

// Start starts an Instance of gotronbrowserwindow
func (gbw *BrowserWindow) Start(forceInstall ...bool) (isdone chan bool, err error) {
	defer errz.Recover(&err)

	var _forceInstall bool
	for _, v := range forceInstall {
		_forceInstall = v
		break
	}

	isdone = done

	// build up structure
	err = gbw.CreateAppStructure(_forceInstall)
	errz.Fatal(err)

	// run sockets and electron
	err = gbw.runApplication()
	errz.Fatal(err)

	return
}

// CreatAppStructure -
// Get electron and web files. Put them into gbw.AppFolder (default ".gotron")
func (gbw *BrowserWindow) CreateAppStructure(polymer bool, forceInstall ...bool) (err error) {
	var _forceInstall bool
	for _, v := range forceInstall {
		_forceInstall = v
	}
	defer errz.Recover(&err)

	if polymer {
		err = gbw.buildPolymerProject()
		errz.Fatal(err)
	}

	err = os.MkdirAll(gbw.AppDirectory, 0777)
	errz.Fatal(err)

	err = gbw.copyTemplate(_forceInstall)
	errz.Fatal(err)

	// Run npm install
	err = gbw.runNPM(_forceInstall)
	errz.Fatal(err)

	// Copy Electron Files
	err = gbw.copyElectronApplication(_forceInstall)
	errz.Fatal(err)

	return nil
}

func (gbw *BrowserWindow) buildPolymerProject() (err error) {
	/*installPolymerCli := exec.Command("npm", "install","polymer-cli")
	installPolymerCli.Stdout = os.Stdout
	installPolymerCli.Stderr = os.Stderr
	installPolymerCli.Dir = gbw.UIFolder
	err = installPolymerCli.Start()
	errz.Fatal(err)
	err = installPolymerCli.Wait()
	errz.Fatal(err)*/
	polymerBuild := exec.Command("polymer",
		"build",
		"--name=es6-bundled", "--preset=es2016",
		"--js-transform-modules-to-amd",
		"--module-resolution=node",
		"--js-minify",
		"--css-minify",
		"--html-minify",
		"--bundle",
		"--add-service-worker",
		"--npm",
	)
	polymerBuild.Stdout = os.Stdout
	polymerBuild.Stderr = os.Stderr
	polymerBuild.Dir = gbw.UIFolder
	err = polymerBuild.Start()
	errz.Fatal(err)
	err = polymerBuild.Wait()
	errz.Fatal(err)
	gbw.UIFolder = filepath.Join(gbw.UIFolder, "build", "es6-bundled")
	return err
}

// runApplication starts websockets and runs the electron application
func (gbw *BrowserWindow) runApplication() (err error) {
	//run websocket
	gbw.runWebsocket()

	//get electron start parameters
	electronPath, args, err := gbw.createStartParameters()
	errz.Fatal(err)

	//run electron
	electron := exec.Command(electronPath, args...)

	electron.Stdout = os.Stdout
	electron.Stderr = os.Stderr

	err = electron.Start()
	errz.Fatal(err)

	gbw.Running = true

	return
}

func (gbw *BrowserWindow) copyTemplate(forceInstall bool) (err error) {
	// Copy app Directory
	mainJS := filepath.Join(gbw.AppDirectory, "main.js")
	firstRun := !file.Exists(mainJS)

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("No caller information")
	}
	gbwDirectory := filepath.Dir(filename)

	if firstRun || forceInstall {
		templateDir := filepath.Join(gbwDirectory, templateApplicationDir)
		err = copy.Copy(templateDir, gbw.AppDirectory)
		errz.Fatal(err)
	}
	err = os.Chmod(gbw.AppDirectory, gotronFileMode)
	errz.Fatal(err)
	return
}

// copyElectronApplication from library package to defined app directory.
// copy app files (.js .css) to app directory
//
// forceInstall forces a reinstallation of electron
// and resets AppDirectory/assets if no UIFolder was set.
//
// On the first run we copy a default application
// into AppDirectory and install electronjs locally.
// When a ui directory was set we use the contents of those
// and copy it into AppDirectory/assets
func (gbw *BrowserWindow) copyElectronApplication(forceInstall bool) (err error) {
	defer errz.Recover(&err)

	/*assetsDir := filepath.Join(gbw.AppDirectory, "assets")
	err = os.Chmod(assetsDir, gotronFileMode)
	errz.Fatal(err)*/

	// If no UI folder is set use default ui files
	if gbw.UIFolder == "" {
		return
	}

	// UIFolder must contain a index.htm(l)
	html := filepath.Join(gbw.UIFolder, "index.html")
	htm := filepath.Join(gbw.UIFolder, "index.htm")
	if !(file.Exists(html) || file.Exists(htm)) {
		return fmt.Errorf("index.htm(l) missing in %s", gbw.UIFolder)
	}

	// No need to copy web application files
	// when no ui folder is set.
	// Also check for ".gotron/assets". This is the
	// default directory when called from gotron-builder,
	// avoids deleting asset dir by accident.
	src, err := filepath.Abs(gbw.UIFolder)
	errz.Fatal(err)
	dst, err := filepath.Abs(filepath.Join(gbw.AppDirectory))
	errz.Fatal(err)

	if src != dst {
		/*err = os.RemoveAll(filepath.Join(gbw.AppDirectory, "assets"))
		errz.Fatal(err)*/

		err = copy.Copy(src, dst)
		errz.Fatal(err)
	}

	return nil
}

func compareVersion(v1, v2 string) string {
	v1 = strings.TrimSpace(v1)
	v2 = strings.TrimSpace(v2)
	if v1 == "" && v2 != "" {
		return v2
	}
	if v1 != "" && v2 == "" {
		return v1
	}
	if v1 == "latest" || v2 == "latest" {
		return "latest"
	}
	v1 = strings.TrimPrefix(v1, "^")
	v2 = strings.TrimPrefix(v2, "^")
	v1s := strings.Split(v1, ".")
	v2s := strings.Split(v2, ".")
	verNum := func(vs []string, i int) int {
		if len(vs) >= i {
			return 0
		}
		n, err := strconv.Atoi(vs[i])
		errz.Fatal(err)
		return n
	}
	for i := 0; i < len(v1) && i < len(v2); i++ {
		if verNum(v1s, i) > verNum(v2s, i) {
			return v1
		} else if verNum(v1s, i) < verNum(v2s, i) {
			return v2
		}
	}
	return v1
}

type packageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Main            string            `json:"main"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

//runNPM - run npm install if not done already or foced.
func (gbw *BrowserWindow) runNPM(forceinstall bool) (err error) {
	defer errz.Recover(&err)

	nodeModules := filepath.Join(gbw.AppDirectory, "node_modules/")
	forceinstall = !file.Exists(nodeModules)

	if forceinstall {
		logger.Debug().Msgf("Installing npm packages...")

		cmd := exec.Command("npm", "install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = gbw.AppDirectory
		err = cmd.Start()

		errz.Fatal(err)

		logger.Debug().Msgf("Waiting for batch")

		err = cmd.Wait()
		errz.Fatal(err)
		logger.Debug().Msgf("Batch done")
	}
	return err
}

//runWebsocket with defined port or look for free port if taken
func (gbw *BrowserWindow) runWebsocket() {
	var err error
	errz.Recover(&err)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(gbw.Port))
	errz.Fatal(err)

	logger.Debug().Msgf("Using port: %d", listener.Addr().(*net.TCPAddr).Port)
	gbw.Port = listener.Addr().(*net.TCPAddr).Port

	// Endpoint for Electron startup/teardown
	// + browser window events to nodejs
	http.HandleFunc("/browser/window/events", gbw.mainEventSocket)
	// Endpoint for ipc like messages
	// send from user web application
	http.HandleFunc("/web/app/events", gbw.onSocket)
	go http.Serve(listener, nil) // Start websockets in goroutine

}

//createStartParameters returns absolute electron path and list of arguments to be passed on electron run call.
func (gbw *BrowserWindow) createStartParameters() (electronPath string, arguments []string, err error) {
	defer errz.Recover(&err)

	electronPath, err = filepath.Abs(filepath.Join(gbw.AppDirectory + "/node_modules/.bin/electron"))
	errz.Fatal(err)
	appPath, err := filepath.Abs(gbw.AppDirectory + "main.js")
	errz.Fatal(err)
	logger.Debug().Msgf(appPath)

	configString, err := json.Marshal(gbw.WindowOptions)
	errz.Fatal(err)

	arguments = []string{appPath, strconv.Itoa(gbw.Port), string(configString)}

	return
}
