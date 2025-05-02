package sy50updater

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"github.com/Microsoft/go-winio"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"bufio"
  "log"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils/rpc"
)

var (
	Sy50Updater      = resource.NewModel("walicki", "sy50updater", "SY50-updater")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(sensor.API, Sy50Updater,
		resource.Registration[sensor.Sensor, *sensorConfig]{
			Constructor: newSy50updaterSy50Updater,
		},
	)
}

type sensorConfig struct {
	DownloadURL   string `json:"download_url"`
	VersionTarget string `json:"version_target"`
	UserPrompt    bool   `json:"prompt"`
}

// Validate ensures all parts of the config are valid and important fields exist.
// Returns implicit dependencies based on the config.
// The path is the JSON path in your robot's config (not the `Config` struct) to the
// resource being validated; e.g. "components.0".
func (cfg *sensorConfig) Validate(path string) ([]string, error) {
	// Add config validation code here
	return nil, nil
}

type sy50updaterSy50Updater struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *sensorConfig

	cancelCtx  context.Context
	cancelFunc func()
}


func newSy50updaterSy50Updater(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[*sensorConfig](rawConf)
	if err != nil {
		return nil, err
	}

	return NewSy50Updater(ctx, deps, rawConf.ResourceName(), conf, logger)
}


func NewSy50Updater(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *sensorConfig, logger logging.Logger) (sensor.Sensor, error) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &sy50updaterSy50Updater{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	return s, nil
}


func (s *sy50updaterSy50Updater) Name() resource.Name {
	return s.name
}


func (s *sy50updaterSy50Updater) NewClientFromConn(ctx context.Context, conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) (sensor.Sensor, error) {
	return nil, errUnimplemented
}


func (s *sy50updaterSy50Updater) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	// Find the Simrad SY50 version in the Windows Registry
	programName := "Simrad SY50"
	version, err := getWindowsProgramVersion(programName)
	if err != nil {
		// Not installed or not found
		version = "Not installed"
	}

	s.logger.Infof("Simrad SY50 version details: %s", version)
	return map[string]interface{}{
		"Simrad SY50 version": version,
	}, nil
}


func (s *sy50updaterSy50Updater) Close(context.Context) error {
	// Put close code here
	s.cancelFunc()
	return nil
}


func (s *sy50updaterSy50Updater) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	var strCaption string
	var strText string
	var invokeWindowsUpdater = false

	// Find the Simrad SY50 version in the Windows Registry
	programName := "Simrad SY50"
	version, err := getWindowsProgramVersion(programName)
	if err != nil {
		// Simrad SY50 not installed or not found in Windows Registry
		// Ask the user if they would like to download and install the latest Simrad SY50 %s update
		version = ""
		invokeWindowsUpdater = true
	} else {
		s.logger.Infof("Comparing Simrad SY50 installed version %s with desired version %s", version, s.cfg.VersionTarget)

		// Compare the currently installed version of Simrad SY50 to the target version
		if IsVersionTargetGreater(version, s.cfg.VersionTarget) {
			// Tell user that Simrad SY50 %s is installed but an %s update is available.
			// Ask if they would like to download and install the latest Simrad SY50 update.
			invokeWindowsUpdater = true
		} else {
			// No Simrad SY50 update required"
			// Simrad SY50 %s is installed and is the current version. No upgrade is required.
		}
		s.logger.Info(strCaption)
		s.logger.Info(strText)
	}

	if invokeWindowsUpdater {
		var response string
		response = "Yes\n"
		if s.cfg.UserPrompt == true {
			// This is not a silent install, interactively ask the user if the Simrad software should be updated
			// The Viam sy50updater module is running as a background service and cannot interact with the desktop
			// Spawn a process in the foreground, PopUp a MessageBox
		  err := SpawnProcess("C:\\Users\\Admin\\simradmsgbox.exe", []string{ version, s.cfg.VersionTarget })
			if err != nil {
				s.logger.Info("Failed to spawn the SimradMsgBox process: %v", err)
				// return here with a failure
			}
			s.logger.Info("simradmsgbox.exe process launched in Session 1 foreground")

			// Wait for the user to respond, pass the response via a Named Pipe to the background service
			response, err := wait4UserResponse()
			if err != nil {
				s.logger.Info("Failed to connect to SimradMsgBox process: %v", err)
				// return here with a failure
			}
			// The user may have responded to the MessageBox "No\n" or "Cancel\n"
			s.logger.Info(response)
		}

		if response == "Yes\n" {
			// Call the generic windows-updater module DoCommand with the appropriate config attributes
			s.logger.Info("Call the generic windows-updater module DoCommand with the appropriate config attributes")
		}
	}

	return map[string]interface{}{
		"Simrad sy50updater DoCommand exit:": strText,
	}, nil
}


func getWindowsProgramVersion(programName string) (string, error) {
	var subkey string

	// Attempt to find the program's uninstall information in the registry.
	if programName != "" {
		subkey = `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`

		k, err := registry.OpenKey(registry.LOCAL_MACHINE, subkey, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			return "", err
		}
		defer k.Close()

		subkeys, err := k.ReadSubKeyNames(-1)
		if err != nil {
			return "", err
		}

		for _, name := range subkeys {
			appKeyPath := filepath.Join(subkey, name)
			appKey, err := registry.OpenKey(registry.LOCAL_MACHINE, appKeyPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			defer appKey.Close()

			displayName, _, err := appKey.GetStringValue("DisplayName")
			if err != nil {
				continue
			}

			if strings.Contains(displayName, programName) {
				version, _, err := appKey.GetStringValue("DisplayVersion")
				if err != nil {
					return "", err
				}
				return version, nil
			}
		}
	}
	return "", fmt.Errorf("program '%s' not found or version information unavailable", programName)
}


// IsVersionTargetGreater compares two version strings and returns true if VersionTarget > VersionCurrent
func IsVersionTargetGreater(VersionCurrent, VersionTarget string) bool {
	currentVerParts := strings.Split(VersionCurrent, ".")
	targetVerParts := strings.Split(VersionTarget, ".")

	// Compare each segment numerically
	for i := 0; i < len(currentVerParts) && i < len(targetVerParts); i++ {
		currentNum, err1 := strconv.Atoi(strings.TrimLeft(currentVerParts[i], "0"))
		targetNum,  err2 := strconv.Atoi(strings.TrimLeft(targetVerParts[i], "0"))

		// Handle errors or empty segments as zero
		if err1 != nil {
			currentNum = 0
		}
		if err2 != nil {
			targetNum = 0
		}

		if targetNum > currentNum {
			return true
		} else if targetNum < currentNum {
			return false
		}
	}

	// If all segments are equal up to the length of the shorter one, check remaining parts
	return len(targetVerParts) > len(currentVerParts)
}


func SpawnProcess(appPath string, args []string) error {
	// Step 1: Get active session ID
	sessionID := windows.WTSGetActiveConsoleSessionId()

	// Step 2: Obtain user token from active session
	var userToken windows.Token
	err := windows.WTSQueryUserToken(sessionID, &userToken)
	if err != nil {
		return fmt.Errorf("WTSQueryUserToken failed: %w", err)
	}
	defer userToken.Close()

	// Step 3: Duplicate token for CreateProcessAsUser
	var duplicatedToken windows.Token
	err = windows.DuplicateTokenEx(
		userToken,
		windows.MAXIMUM_ALLOWED,
		nil,
		windows.SecurityIdentification,
		windows.TokenPrimary,
		&duplicatedToken,
	)
	if err != nil {
		return fmt.Errorf("DuplicateTokenEx failed: %w", err)
	}
	defer duplicatedToken.Close()

	// Step 4: Set up startup info with WinSta0\Default desktop
	var startupInfo windows.StartupInfo
	startupInfo.Cb = uint32(unsafe.Sizeof(startupInfo))
	startupInfo.Desktop = syscall.StringToUTF16Ptr("winsta0\\default")

	var procInfo windows.ProcessInformation

	cmdLine := syscall.StringToUTF16Ptr(appPath + " " + strings.Join(args, " ") )

	// Step 5: Create process as user in active session
	err = windows.CreateProcessAsUser(
		duplicatedToken,
		nil,
		cmdLine,
		nil,
		nil,
		false,
		0,
		nil,
		nil,
		&startupInfo,
		&procInfo,
	)
	if err != nil {
		return fmt.Errorf("CreateProcessAsUser failed: %w", err)
	}

	defer windows.CloseHandle(procInfo.Thread)
	defer windows.CloseHandle(procInfo.Process)

	return nil
}


const pipeName = `\\.\pipe\viam_sy50_pipe`

func wait4UserResponse() (string, error) {
	var response string
	pipeCfg := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;WD)", // Allow all access (for demo only!)
		InputBufferSize:    128,
		OutputBufferSize:   128,
	}

	listener, err := winio.ListenPipe(pipeName, pipeCfg)
	if err != nil {
		return "",fmt.Errorf("Failed to create named pipe: %v", err)
	}
	defer listener.Close()
	//log.Printf("Viam Sy50updater module listening on Named Pipe %s\n", pipeName)

	conn, err := listener.Accept()
	if err != nil {
		return "",fmt.Errorf("Accept failed: %v", err)
	}
	defer conn.Close()
	//log.Println("Client connected")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		response := scanner.Text()
		log.Printf("Received from client: %s", response)
		//_, _ = conn.Write([]byte("Echo: " + line + "\n"))
	}

	if err := scanner.Err(); err != nil {
		return "",fmt.Errorf("Read error: %v", err)
	}
	//log.Println("Client disconnected")
	return response, nil
}
